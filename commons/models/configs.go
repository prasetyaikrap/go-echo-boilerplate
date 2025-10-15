package models

import (
	"time"

	"github.com/labstack/echo/v4/middleware"
)

type ApplicationConfig struct {
	Port 					string
	ClientID				string
	AllowedCleintIDs 		[]string
	SecretToken				string
	JWTAccessSecret			string
	JWTRefreshSecret		string
	AccessTokenExpiration 	time.Duration
	RefreshTokenExpiration 	time.Duration
	CORSConfig				middleware.CORSConfig
}

type DBConfig struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     string
	TimeZone string
	SSLMode  string 
}

type ENVConfig struct {
	Application 	ApplicationConfig
	DB			DBConfig
}