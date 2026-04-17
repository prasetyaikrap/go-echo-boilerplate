package main

import (
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
	"go-serviceboilerplate/interfaces/http/validator"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Boilerplate Go Echo API Service
// @version 1.0
// @description Boilerplate Go Echo API Service
func main() {
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
	e.Use(slogLoggerMiddleware)
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.CORSWithConfig(configs.Envs.Application.CORSConfig))
	e.Use(authMiddleware.VerifyClient)

	e.Validator = validator.NewCustomValidator()
	e.HTTPErrorHandler = utils.HttpErrorHandler

	// System Routes
	systemRoutes := e.Group("/system")
	systemHandler.RegisterRoutes(systemRoutes)

	// Swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", configs.Envs.Application.Port)))
}