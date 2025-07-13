package postgres

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

type PostgresInstance struct {
	DB 			*gorm.DB
}

func NewPostgressInstance(configs *configurations.Configs) *PostgresInstance {
	mainDB := InitDatabase(configs, false)

	return &PostgresInstance{DB: mainDB}
}

func InitDatabase(configs *configurations.Configs, autoMigrate bool) (DB *gorm.DB) {
	envConfigs := configs.Env	
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		envConfigs.DB.Host,
		envConfigs.DB.User,
		envConfigs.DB.Password,
		envConfigs.DB.DBName,
		envConfigs.DB.Port,
		envConfigs.DB.SSLMode,
		envConfigs.DB.TimeZone,
	)

	// Configure GORM logger for better visibility in development/production
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Info,   // Log level: Silent, Error, Warn, Info
			ParameterizedQueries:      true,          // Log parameterized queries
			Colorful:                  true,          // Enable color for log output
		},
	)

	gormConfig := &gorm.Config{
		Logger: newLogger,
	}
	
	DB, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
        log.Fatalf("failed to connect to database: %v", err)
    }

	sqlDB, err := DB.DB()
	if err != nil {
        log.Fatalf("failed to get underlying sql.DB: %v", err)
    }

	sqlDB.SetMaxIdleConns(10)                  // Maximum number of idle connections in the pool
	sqlDB.SetMaxOpenConns(100)                 // Maximum number of open connections to the database
	sqlDB.SetConnMaxLifetime(5 * time.Minute)  // Maximum amount of time a connection may be reused

	fmt.Println("Database Successfully Connected")

	if(autoMigrate) {
		AutoMigrate(DB)
	}

	return DB
}

func AutoMigrate(db *gorm.DB, dst ...interface{}) {
	err := db.AutoMigrate(dst)

	if(err != nil) {
		log.Fatalf("AutoMigrate failed: %v", err)
	} 

	fmt.Println("Database Migration Successful")
}