package configurations

import (
	"go-serviceboilerplate/commons/models"
)

type Configs struct {
	Envs *models.ENVConfig
	Logger *SlogLogger
}

func NewConfigurations() *Configs {
	envConfigs := GetENVConfig()
	logger := NewSlogLogger()
	return &Configs{
		Envs:  envConfigs,
		Logger: logger,
	}
}