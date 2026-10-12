package postgres

import (
	"fmt"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewPostgresInstance opens a Postgres connection. Create one per database the service needs, e.g.
//
//	mainDB := postgres.NewPostgresInstance(configs, configs.Envs.MainDB, &entities.User{})
//	analyticsDB := postgres.NewPostgresInstance(configs, configs.Envs.AnalyticsDB, &entities.Event{})
//
// migrateModels are auto-migrated only when dbConfig.AutoMigrate is true
func NewPostgresInstance(configs *configurations.Configs, dbConfig models.PostgresConfig, migrateModels ...any) *gorm.DB {
	db, err := connect(configs, dbConfig)
	if err != nil {
		configs.Logger.Fatal("Database initialization failed", "name", dbConfig.Name, "error", err)
	}

	if dbConfig.AutoMigrate {
		if err := db.AutoMigrate(migrateModels...); err != nil {
			configs.Logger.Fatal("Database migration failed", "name", dbConfig.Name, "error", err)
		}

		configs.Logger.Info("Database Migration Successful", "name", dbConfig.Name)
	}

	return db
}

func connect(configs *configurations.Configs, dbConfig models.PostgresConfig) (*gorm.DB, error) {
	// Configure GORM logger for better visibility in development/production
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,                      // Slow SQL threshold
			LogLevel:                  parseLogLevel(dbConfig.LogLevel), // Log level: Silent, Error, Warn, Info
			IgnoreRecordNotFoundError: true,                             // Ignore ErrRecordNotFound error for logging
			ParameterizedQueries:      true,                             // Log parameterized queries
			Colorful:                  true,                             // Enable color for log output
		},
	)

	gormConfig := &gorm.Config{
		Logger: newLogger,
		TranslateError: true,
	}

	db, err := gorm.Open(postgres.Open(dbConfig.DSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(dbConfig.MaxConnIdle)            // Maximum number of idle connections in the pool
	sqlDB.SetConnMaxIdleTime(dbConfig.MaxConnIdleLifeTime) // Maximum amount of time a connection may be idle
	sqlDB.SetMaxOpenConns(dbConfig.MaxConn)                // Maximum number of open connections to the database
	sqlDB.SetConnMaxLifetime(dbConfig.MaxConnLifeTime)     // Maximum amount of time a connection may be reused

	configs.Logger.Info("Database Successfully Connected", "name", dbConfig.Name)

	return db, nil
}

func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	default:
		return logger.Info
	}
}
