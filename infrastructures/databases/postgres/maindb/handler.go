package maindb

import (
	"fmt"
	"go-serviceboilerplate/infrastructures/configurations"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type AuthPostgresInstance struct {
	configs *configurations.Configs
}

func NewAuthPostgressInstance(configs *configurations.Configs) *gorm.DB {
	instance := &AuthPostgresInstance{
		configs: configs,
	}
	db, err := instance.Database()
	if err != nil {
        instance.configs.Logger.Fatal("Database initialization failed", err)
    }

	instance.AutoMigrate(db)
	
	return db
}

func (a *AuthPostgresInstance) Database() (DB *gorm.DB, err error)  {	
	// Configure GORM logger for better visibility in development/production
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Info,   // Log level: Silent, Error, Warn, Info
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logging
			ParameterizedQueries:      true,          // Log parameterized queries
			Colorful:                  true,          // Enable color for log output
		},
	)

	gormConfig := &gorm.Config{
		Logger: newLogger,
		TranslateError: true,
	}
	
	DB, err = gorm.Open(postgres.Open(a.configs.Envs.DB.DSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(a.configs.Envs.DB.MaxConnIdle) 
	sqlDB.SetConnMaxIdleTime(a.configs.Envs.DB.MaxConnIdleLifeTime)         // Maximum number of idle connections in the pool
	sqlDB.SetMaxOpenConns(a.configs.Envs.DB.MaxConn)          // Maximum number of open connections to the database
	sqlDB.SetConnMaxLifetime(a.configs.Envs.DB.MaxConnLifeTime)  	// Maximum amount of time a connection may be reused

	a.configs.Logger.Info("Database Successfully Connected")

	return DB, nil
}

func (a *AuthPostgresInstance) AutoMigrate(db *gorm.DB) {
	err := db.AutoMigrate()

	if(err != nil) {
		log.Fatalf("AutoMigrate failed: %v", err)
	} 

	a.configs.Logger.Info("Database Migration Successful")
}