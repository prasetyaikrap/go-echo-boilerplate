package usecases

import (
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastrucutres/repositories"
	"go-serviceboilerplate/infrastrucutres/security"
)

type AuthenticationsUsecase struct {
	AuthenticationsRepository *repositories.AuthenticationsRepositories
	TokenManagerSecurity *security.TokenManagerSecurity
	PasswordHashSecurity *security.PasswordHashSecurity
}

func NewAuthenticationsUsecase(authenticationsRepository *repositories.AuthenticationsRepositories,  tokenManagerSecurity *security.TokenManagerSecurity, passwordHashSecurity *security.PasswordHashSecurity) *AuthenticationsUsecase {
	return &AuthenticationsUsecase{authenticationsRepository, tokenManagerSecurity, passwordHashSecurity}
}

func (r *AuthenticationsUsecase) VerifyClient(clientID string) error {
	clientIDerr := r.AuthenticationsRepository.VerifyClientID(clientID)
	if(clientIDerr != nil) {
		return clientIDerr
	}

	return nil
}

func (r *AuthenticationsUsecase) VerifySecretToken(secretToken string) error {
	secretTokenErr := r.AuthenticationsRepository.VerifySecretToken(secretToken)
	if(secretTokenErr != nil) {
		return secretTokenErr
	}

	return nil
}

func (r *AuthenticationsUsecase) VerifyAccessToken(bearerToken string) (*models.AccessTokenClaims, error ){
	authToken, authTokenErr := r.AuthenticationsRepository.VerifyAuthToken(bearerToken)
	if(authTokenErr != nil) {
		return nil, authTokenErr
	}

	userClaim, claimErr := r.TokenManagerSecurity.VerifyAccessToken(authToken, false)
	if(claimErr != nil) {
		return nil, claimErr
	}

	return userClaim, nil
}

func (r *AuthenticationsUsecase) VerifyRefreshToken(bearerToken string) (*models.RefreshTokenClaims, error ){
	authToken, authTokenErr := r.AuthenticationsRepository.VerifyAuthToken(bearerToken)
	if(authTokenErr != nil) {
		return nil, authTokenErr
	}

	userClaim, claimErr := r.TokenManagerSecurity.VerifyRefreshToken(authToken, false)
	if(claimErr != nil) {
		return nil, claimErr
	}

	return userClaim, nil
}