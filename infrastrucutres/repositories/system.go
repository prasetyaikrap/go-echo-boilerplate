package repositories

import "gorm.io/gorm"

type SystemRepositories struct {
	postgresDB *gorm.DB
}

func NewSystemRepositories(postgresDB *gorm.DB) *SystemRepositories {
	return &SystemRepositories{postgresDB}
}

func (s *SystemRepositories) GetSystemInfo() map[string]string {
	data := map[string]string{
		"app_name": "Go Service Boilerplate",
		"version":  "1.0.0",
		"status":   "running",
	}

	return data
}