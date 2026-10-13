package main

import (
	"context"
	"errors"
	"fmt"
	"go-serviceboilerplate/applications/usecases"
	"go-serviceboilerplate/commons/utils"
	_ "go-serviceboilerplate/docs"
	"go-serviceboilerplate/infrastructures/configurations"
	"go-serviceboilerplate/infrastructures/databases"
	"go-serviceboilerplate/infrastructures/repositories"
	"go-serviceboilerplate/infrastructures/security"
	"go-serviceboilerplate/interfaces/http/api/health"
	"go-serviceboilerplate/interfaces/http/api/system"
	authMiddleware "go-serviceboilerplate/interfaces/http/middlewares/authentications"
	loggerMiddleware "go-serviceboilerplate/interfaces/http/middlewares/logger"
	panicRecoveryMiddleware "go-serviceboilerplate/interfaces/http/middlewares/panic_recovery"
	requestContextMiddleware "go-serviceboilerplate/interfaces/http/middlewares/request_context"
	requestLimitMiddleware "go-serviceboilerplate/interfaces/http/middlewares/request_limit"
	"go-serviceboilerplate/interfaces/http/validator"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Boilerplate Go Echo API Service
// @version 1.0
// @description Boilerplate Go Echo API Service
func main() {
	// os.Exit skips deferred calls, so the app runs in run() and only the exit code is handled here.
	// A non-zero code tells the container runtime (Kubernetes, Docker) that the app failed
	os.Exit(run())
}

func run() int {
	// App context — cancelled on SIGINT/SIGTERM. Pass it to background jobs so they stop on shutdown.
	// HTTP requests do not derive from it: on shutdown in-flight requests are allowed to finish (SHUTDOWN_TIMEOUT)
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Configuration
	configs := configurations.NewConfigurations()

	// Initialize database instances
	dbInstances := databases.NewDatabaseInstance(configs)

	// Security
	passwordHashSecurity := security.NewPasswordHashSecurity(configs, 10)
	tokenManagerSecurity := security.NewTokenManagerSecurity(configs)

	// Repositories
	systemRepositories := repositories.NewSystemRepositories(dbInstances)
	authenticationsRepositories := repositories.NewAuthenticationsRepositories(dbInstances, configs)

	// Usecases
	systemUsecase := usecases.NewSystemUsecase(systemRepositories)
	authenticationsUsecase := usecases.NewAuthenticationsUsecase(authenticationsRepositories, tokenManagerSecurity, passwordHashSecurity)

	// Handlers
	systemHandler := system.NewSystemHandler(systemUsecase)
	// Readiness checks: add every dependency the service needs to serve traffic (e.g. "redis": mainRedis.Ping)
	healthHandler := health.NewHealthHandler(configs, map[string]health.ReadinessCheck{
		"database": dbInstances.Ping,
	})

	// Middlewares & Misc
	authMiddleware := authMiddleware.NewAuthMiddlewareHandler(authenticationsUsecase)
	slogLoggerMiddleware := loggerMiddleware.NewSlogLoggerMiddleware(configs)

	// Routes
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true // startup is logged through slog
	// Order matters: the request ID must exist before logging, CORS must run before anything that can reject a request
	// (so browsers can read the error), and the timeout must wrap everything that does work
	e.Use(requestContextMiddleware.NewRequestIDMiddleware())
	e.Use(slogLoggerMiddleware)
	e.Use(panicRecoveryMiddleware.NewPanicRecoveryMiddleware())
	e.Use(echoMiddleware.CORSWithConfig(configs.Envs.Application.CORSConfig))
	e.Use(requestLimitMiddleware.NewBodyLimitMiddleware(configs))
	e.Use(requestContextMiddleware.NewRequestTimeoutMiddleware(configs))
	e.Use(authMiddleware.VerifyClient)

	e.Validator = validator.NewCustomValidator()
	e.HTTPErrorHandler = utils.HttpErrorHandler

	// Health probes (/healthz, /readyz) — no X-Client-Id required
	healthHandler.RegisterRoutes(e)

	// System Routes
	systemRoutes := e.Group("/system")
	systemHandler.RegisterRoutes(systemRoutes)

	// Swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Connection-level timeouts. ReadHeaderTimeout is on by default; the others are opt-in via env (see .env_example)
	e.Server.ReadHeaderTimeout = configs.Envs.Application.ServerReadHeaderTimeout
	e.Server.ReadTimeout = configs.Envs.Application.ServerReadTimeout
	e.Server.WriteTimeout = configs.Envs.Application.ServerWriteTimeout
	e.Server.IdleTimeout = configs.Envs.Application.ServerIdleTimeout

	// e.Start blocks until the server stops, so it runs in a goroutine while run() waits for a shutdown signal.
	// A startup or runtime failure is sent back here so it goes through the same cleanup as a normal shutdown
	serverErr := make(chan error, 1)
	go func() {
		configs.Logger.Info("HTTP server starting", "port", configs.Envs.Application.Port)

		if err := e.Start(fmt.Sprintf(":%s", configs.Envs.Application.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	exitCode := 0

	select {
	case <-appCtx.Done():
		configs.Logger.Info("Shutdown signal received, stopping gracefully")
	case err := <-serverErr:
		configs.Logger.Error("HTTP server failed, shutting down", "error", err)
		exitCode = 1
	}

	stop() // a second Ctrl+C now terminates immediately

	// Not derived from appCtx (already cancelled): gives in-flight requests up to SHUTDOWN_TIMEOUT to finish
	shutdownCtx := context.Background()
	if timeout := configs.Envs.Application.ShutdownTimeout; timeout > 0 {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(shutdownCtx, timeout)
		defer cancel()
	}

	// Stop accepting requests and wait for in-flight ones, then close the resources they use
	if err := e.Shutdown(shutdownCtx); err != nil {
		configs.Logger.Error("HTTP server shutdown failed", "error", err)
		exitCode = 1
	}

	if err := dbInstances.Close(); err != nil {
		configs.Logger.Error("Database close failed", "error", err)
		exitCode = 1
	}

	configs.Logger.Info("Shutdown complete", "exit_code", exitCode)

	return exitCode
}
