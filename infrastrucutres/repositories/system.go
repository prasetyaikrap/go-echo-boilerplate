package repositories

import (
	"go-serviceboilerplate/infrastrucutres/databases/postgres"
)

type SystemRepositories struct {
	postgresDB *postgres.PostgresInstance
}

func NewSystemRepositories(postgresDB *postgres.PostgresInstance) *SystemRepositories {
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