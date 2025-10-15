package main

import (
	"fmt"
	"go-serviceboilerplate/applications/usecases"
	_ "go-serviceboilerplate/docs"
	"go-serviceboilerplate/infrastrucutres/configurations"
	"go-serviceboilerplate/infrastrucutres/databases/postgres/maindb"
	"go-serviceboilerplate/infrastrucutres/repositories"
	"go-serviceboilerplate/infrastrucutres/security"
	"go-serviceboilerplate/interfaces/http/api/system"
	authMiddleware "go-serviceboilerplate/interfaces/http/middlewares/authentications"
	loggerMiddleware "go-serviceboilerplate/interfaces/http/middlewares/logger"
	"go-serviceboilerplate/interfaces/http/validator"
	"go-serviceboilerplate/interfaces/utils"

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

	// Initialize Postgres database connection
	mainDb := maindb.NewAuthPostgressInstance(configs)

	// Security
	passwordHashSecurity := security.NewPasswordHashSecurity(configs, 10)
	tokenManagerSecurity := security.NewTokenManagerSecurity(configs)

	// Repositories
	systemRepositories := repositories.NewSystemRepositories(mainDb)
	authenticationsRepositories := repositories.NewAuthenticationsRepositories(mainDb, configs)

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