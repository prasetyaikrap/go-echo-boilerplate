package repositories

import (
	"gorm.io/gorm"
)

type SystemRepositories struct {
	mainDB *gorm.DB
}

func NewSystemRepositories(mainDB *gorm.DB) *SystemRepositories {
	return &SystemRepositories{mainDB}
}

func (s *SystemRepositories) GetSystemInfo() map[string]string {
	data := map[string]string{
		"app_name": "Go Service Boilerplate",
		"version":  "1.0.0",
		"status":   "running",
	}

	return data
}