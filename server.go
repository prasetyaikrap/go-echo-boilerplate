package main

import (
	"fmt"
	"go-serviceboilerplate/applications/usecases"
	_ "go-serviceboilerplate/docs"
	"go-serviceboilerplate/infrastrucutres/configurations"
	"go-serviceboilerplate/infrastrucutres/databases/postgres"
	"go-serviceboilerplate/infrastrucutres/repositories"
	"go-serviceboilerplate/interfaces/http/api/system"
	"go-serviceboilerplate/interfaces/http/middlewares"
	"go-serviceboilerplate/interfaces/http/validator"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Boilerplate Go Echo API Service
// @version 1.0
// @description Boilerplate Go Echo API Service
func main() {
	// Configuration
	configs := configurations.NewConfigurations()

	// Initialize Postgres database connection
	postgres := postgres.NewPostgressInstance(configs)

	// Repositories
	systemRepositories := repositories.NewSystemRepositories(postgres)

	// Usecases
	systemUsecase := usecases.NewSystemUsecase(systemRepositories)
	
	// Handlers
	systemHandler := system.NewSystemHandler(systemUsecase)

	// Middlewares & Misc
	appMiddlewares := middlewares.NewAppMiddlewaresHandler()

	// Routes
	e := echo.New()
	e.Validator = validator.NewCustomValidator()

	systemHandler.RegisterRoute(e, appMiddlewares)

	// Swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", configs.Env.Application.Port)))
}