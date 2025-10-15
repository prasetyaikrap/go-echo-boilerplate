package repositories

import (
	"errors"
	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastrucutres/configurations"
	"slices"
	"strings"

	"gorm.io/gorm"
)

type AuthenticationsRepositories struct {
	mainDB *gorm.DB
	Configs *configurations.Configs
}

func NewAuthenticationsRepositories(mainDB *gorm.DB, configs *configurations.Configs) *AuthenticationsRepositories {
	return &AuthenticationsRepositories{mainDB, configs}
}

func (s AuthenticationsRepositories) VerifyClientID(clientID string) error {
	envConfigs := s.Configs.Envs
	isAllClientIDsAllowed := slices.Contains(envConfigs.Application.AllowedCleintIDs, "*")
	isClientIDAllowed := slices.Contains(envConfigs.Application.AllowedCleintIDs, clientID)
	
	if clientID == "" ||  (!isAllClientIDsAllowed && !isClientIDAllowed) {
		return utils.NewAuthenticationError(errors.New("Client not allowed"))
	}

	return nil
}

func (s AuthenticationsRepositories) VerifySecretToken(token string) error {
	envConfigs := s.Configs.Envs
	if token == "" ||  token != envConfigs.Application.SecretToken {
		return utils.NewAuthenticationError(errors.New("Service not allowed"))
	}

	return nil
}

func (s AuthenticationsRepositories) VerifyAuthToken(bearerToken string) (string, error) {
	if !strings.HasPrefix(bearerToken, "Bearer ") {
		return "", utils.NewAuthenticationError(errors.New("invalid Authorization format"))
	}

	token := strings.TrimPrefix(bearerToken, "Bearer ")
	if token == "" {
		return "", utils.NewAuthenticationError(errors.New("invalid Authorization"))
	}
	
	return token, nil
}