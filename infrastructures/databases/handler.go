package databases

import (
	"go-serviceboilerplate/infrastructures/configurations"
	"go-serviceboilerplate/infrastructures/databases/postgres"

	"gorm.io/gorm"
)

// DatabaseInstance holds every database the service connects to. Add a field per extra database
type DatabaseInstance struct {
	MainDB *gorm.DB
}

func NewDatabaseInstance(configs *configurations.Configs) *DatabaseInstance {
	// Register the models to auto-migrate for each database here
	mainDB := postgres.NewPostgresInstance(configs, configs.Envs.MainDB)

	return &DatabaseInstance{MainDB: mainDB}
}
