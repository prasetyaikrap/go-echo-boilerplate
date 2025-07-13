package configurations

import (
	"go-serviceboilerplate/commons/models"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Configs struct {
	Env 	*models.ENVConfig
}

func NewConfigurations() *Configs {
	envConfigs := GetENVConfig()

	return &Configs{Env: envConfigs}
}

func GetENVConfig() *models.ENVConfig {
	if _, err := os.Stat(".env"); err == nil {
		err = godotenv.Load()
		if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}
	clientID := GetEnv("CLIENT_ID", true)

	envConfig := models.ENVConfig{
		Application: models.ApplicationConfig{
			Port: GetEnv("PORT", true),
			ClientID: clientID,
			SecretToken: GetEnv("SERVICE_TOKEN", true),
			JWTSecret: GetEnv("JWT_SECRET", true),
			CORSConfig: middleware.CORSConfig{
				AllowOrigins:     GetAllowedOrigins(),
				AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
				AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization, "X-Client-Id", "X-Service-Token"},
				AllowCredentials: true,
			  },
		},
		DB: models.DBConfig{
			Host:     GetEnv("DB_HOST", true),
			User:     GetEnv("DB_USER", true),
			Password: GetEnv("DB_PASSWORD", true),
			DBName:   GetEnv("DB_NAME", true),
			Port:     GetEnv("DB_PORT", true),
			TimeZone: GetEnv("DB_TIMEZONE", true),
			SSLMode:  GetEnv("DB_SSLMODE", true),
		},
	}
	
	return &envConfig
}

func GetEnv(key string, required bool) string {
	value, exists := os.LookupEnv(key); 

	if(!required) {
		return value
	}

	if(!exists || value == "") {
		log.Fatalf("Environment variable %v is required", key)
	}

	return value
}

func GetAllowedOrigins() []string {
	allowedOrigins := strings.Split(GetEnv("ALLOWED_ORIGINS", false), ",")
	var cleaned []string
	for _, s := range allowedOrigins {
		if s != "" {
			cleaned = append(cleaned, s)
		}
	}

	if(len(cleaned) <= 0) {
		cleaned = append(cleaned, "*")
	}

	return cleaned
}