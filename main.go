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
	"go-serviceboilerplate/interfaces/http/api/system"
	authMiddleware "go-serviceboilerplate/interfaces/http/middlewares/authentications"
	loggerMiddleware "go-serviceboilerplate/interfaces/http/middlewares/logger"
	requestContextMiddleware "go-serviceboilerplate/interfaces/http/middlewares/request_context"
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

	// Middlewares & Misc
	authMiddleware := authMiddleware.NewAuthMiddlewareHandler(authenticationsUsecase)
	slogLoggerMiddleware := loggerMiddleware.NewSlogLoggerMiddleware(configs)

	// Routes
	e := echo.New()
	e.HideBanner = true
	// Order matters: the request ID must exist before logging, and the timeout must wrap everything that does work
	e.Use(requestContextMiddleware.NewRequestIDMiddleware())
	e.Use(slogLoggerMiddleware)
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.CORSWithConfig(configs.Envs.Application.CORSConfig))
	e.Use(requestContextMiddleware.NewRequestTimeoutMiddleware(configs))
	e.Use(authMiddleware.VerifyClient)

	e.Validator = validator.NewCustomValidator()
	e.HTTPErrorHandler = utils.HttpErrorHandler

	// System Routes
	systemRoutes := e.Group("/system")
	systemHandler.RegisterRoutes(systemRoutes)

	// Swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Connection-level timeouts. ReadHeaderTimeout is always on; the others are opt-in via env (see .env_example)
	e.Server.ReadHeaderTimeout = configs.Envs.Application.ServerReadHeaderTimeout
	e.Server.ReadTimeout = configs.Envs.Application.ServerReadTimeout
	e.Server.WriteTimeout = configs.Envs.Application.ServerWriteTimeout
	e.Server.IdleTimeout = configs.Envs.Application.ServerIdleTimeout

	go func() {
		configs.Logger.Info("HTTP server starting", "port", configs.Envs.Application.Port)

		if err := e.Start(fmt.Sprintf(":%s", configs.Envs.Application.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			configs.Logger.Fatal("HTTP server failed", "error", err)
		}
	}()

	<-appCtx.Done()
	stop() // a second Ctrl+C now terminates immediately
	configs.Logger.Info("Shutdown signal received, stopping gracefully")

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
	}

	if err := dbInstances.Close(); err != nil {
		configs.Logger.Error("Database close failed", "error", err)
	}

	configs.Logger.Info("Shutdown complete")
}
