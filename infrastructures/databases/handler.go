package databases

import (
	"go-serviceboilerplate/infrastructures/configurations"
	"go-serviceboilerplate/infrastructures/databases/postgres/maindb"

	"gorm.io/gorm"
)

type DatabaseInstance struct {
	db *gorm.DB
}

func NewDatabaseInstance(configs *configurations.Configs) *DatabaseInstance {
	mainDB := maindb.NewAuthPostgressInstance(configs)

	return &DatabaseInstance{db: mainDB}
}