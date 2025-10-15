package maindb

import (
	"fmt"
	"go-serviceboilerplate/infrastrucutres/configurations"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewAuthPostgressInstance(configs *configurations.Configs) *gorm.DB {
	db, err := Database(configs)
	if err != nil {
        log.Fatalf("Database initialization failed: %v", err)
    }

	AutoMigrate(db)
	
	return db
}

func Database(configs *configurations.Configs) (DB *gorm.DB, err error)  {	
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		configs.Envs.DB.Host,
		configs.Envs.DB.User,
		configs.Envs.DB.Password,
		configs.Envs.DB.DBName,
		configs.Envs.DB.Port,
		configs.Envs.DB.SSLMode,
		configs.Envs.DB.TimeZone,
	)

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
	}
	
	DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)                  // Maximum number of idle connections in the pool
	sqlDB.SetMaxOpenConns(100)                 // Maximum number of open connections to the database
	sqlDB.SetConnMaxLifetime(5 * time.Minute)  // Maximum amount of time a connection may be reused

	fmt.Println("Database Successfully Connected")

	return DB, nil
}

func AutoMigrate(db *gorm.DB) {
	err := db.AutoMigrate()

	if(err != nil) {
		log.Fatalf("AutoMigrate failed: %v", err)
	} 

	fmt.Println("Database Migration Successful")
}