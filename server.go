package main

import (
	"fmt"
	"go-serviceboilerplate/applications/usecases"
	"go-serviceboilerplate/commons/databases/postgres"
	"go-serviceboilerplate/commons/utils"
	_ "go-serviceboilerplate/docs"
	"go-serviceboilerplate/infrastrucutres/repositories"
	"go-serviceboilerplate/interfaces/http/api/system"
	"log"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Boilerplate Go Echo API Service
// @version 1.0
// @description Boilerplate Go Echo API Service
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Initialize Postgres database connection
	DB := postgres.InitPostgres()

	e := echo.New()

	// Repositories
	systemRepositories := repositories.NewSystemRepositories(DB)

	// Usecases
	systemUsecase := usecases.NewSystemUsecase(systemRepositories)
	
	// Handlers
	systemHandler := system.NewSystemHandler(systemUsecase)

	// Routes
	systemHandler.RegisterRoute(e)

	// Swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)


	servicePort := utils.GetEnv("PORT")
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", servicePort)))
}