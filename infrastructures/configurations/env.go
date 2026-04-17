package configurations

import (
	"go-serviceboilerplate/commons/models"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func GetENVConfig() *models.ENVConfig {
	if _, err := os.Stat(".env"); err == nil {
		err = godotenv.Load()
		if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}

	envConfig := models.ENVConfig{
		Application: models.ApplicationConfig{
			Port: GetEnv("PORT", true, ""),
			ClientID: GetEnv("CLIENT_ID", true, ""),
			AllowedCleintIDs: GetAllowedClientIDs(),
			SecretToken: GetEnv("SERVICE_TOKEN", false, ""),
			JWTAccessSecret: GetEnv("JWT_ACCESS_SECRET", false, ""),
			JWTRefreshSecret: GetEnv("JWT_REFRESH_SECRET", false, ""),
			AccessTokenExpiration: time.Duration(GetEnvInt("ACCESS_TOKEN_EXPIRATION", false, 30)) * time.Minute,
			RefreshTokenExpiration: time.Duration(GetEnvInt("REFRESH_TOKEN_EXPIRATION", false, 60 * 24 * 7)) * time.Minute,
			CORSConfig: middleware.CORSConfig{
				AllowOrigins:     GetAllowedOrigins(),
				AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
				AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization, models.XClientIdHeader, models.XServiceTokenHeader, models.XUserIdHeader, models.XRenewTokenHeader},
				AllowCredentials: true,
			  },
		},
		DB: models.DBConfig{
			DSN: 	  GetEnv("DB_DSN", false, ""),
			MaxConnIdle: GetEnvInt("DB_MAX_CONN_IDLE", false, 10),
			MaxConnIdleLifeTime: time.Duration(GetEnvInt("DB_MAX_CONN_IDLE_LIFETIME", false, 30)) * time.Minute,
			MaxConn: GetEnvInt("DB_MAX_CONN", false, 25),
			MaxConnLifeTime: time.Duration(GetEnvInt("DB_MAX_CONN_LIFETIME", false, 60)) * time.Minute,

			AutoMigrate: GetEnv("DB_AUTO_MIGRATE", false, "false") == "true",
		},
	}

	return &envConfig
}

func GetEnv(key string, required bool, defaultValue string) string {
	value, exists := os.LookupEnv(key); 
	if(value == "") {
		value = defaultValue
	}

	if(!required) {
		return value
	}

	if(!exists || value == "") {
		log.Fatalf("Environment variable %v is required", key)
	}

	return value
}

func GetEnvInt(key string, required bool, defaultValue int) int {
	value, exists := os.LookupEnv(key)
	if(value == "") {
		value = "0"
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("Environment variable %v must be a valid integer (got %q)", key, value)
	}

	if(intValue <= 0) {
		intValue = defaultValue
	}
	
	if(!required) {
		return intValue
	}

	if !exists || intValue <= 0 {
		log.Fatalf("Environment variable %v is required", key)
	}

	return intValue
}

func GetAllowedOrigins() []string {
	allowedOrigins := strings.Split(GetEnv("ALLOWED_ORIGINS", false, "*"), ",")
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

func GetAllowedClientIDs() []string {
	allowedClientIDs := strings.Split(GetEnv("ALLOWED_CLIENT_IDS", false, "*"), ",")
	var cleaned []string
	for _, s := range allowedClientIDs {
		if s != "" {
			cleaned = append(cleaned, s)
		}
	}

	if(len(cleaned) <= 0) {
		cleaned = append(cleaned, "*")
	}

	return cleaned
}