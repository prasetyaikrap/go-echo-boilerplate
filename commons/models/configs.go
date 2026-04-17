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
	DSN	  	 string
	MaxConnIdle int
	MaxConnIdleLifeTime time.Duration
	MaxConn int
	MaxConnLifeTime time.Duration

	AutoMigrate bool
}

type ENVConfig struct {
	Application 	ApplicationConfig
	DB			DBConfig
}