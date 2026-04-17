package repositories

import (
	"go-serviceboilerplate/infrastructures/databases"
)

type SystemRepositories struct {
	dbInstances *databases.DatabaseInstance
}

func NewSystemRepositories(dbInstances *databases.DatabaseInstance) *SystemRepositories {
	return &SystemRepositories{dbInstances}
}

func (s *SystemRepositories) GetSystemInfo() map[string]string {
	data := map[string]string{
		"app_name": "Go Service Boilerplate",
		"version":  "1.0.0",
		"status":   "running",
	}

	return data
}