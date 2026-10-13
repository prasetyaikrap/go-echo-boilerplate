package usecases

import (
	"context"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/repositories"
	"go-serviceboilerplate/infrastructures/security"
)

type AuthenticationsUsecase struct {
	AuthenticationsRepository *repositories.AuthenticationsRepositories
	TokenManagerSecurity *security.TokenManagerSecurity
	PasswordHashSecurity *security.PasswordHashSecurity
}

func NewAuthenticationsUsecase(authenticationsRepository *repositories.AuthenticationsRepositories,  tokenManagerSecurity *security.TokenManagerSecurity, passwordHashSecurity *security.PasswordHashSecurity) *AuthenticationsUsecase {
	return &AuthenticationsUsecase{authenticationsRepository, tokenManagerSecurity, passwordHashSecurity}
}

func (r *AuthenticationsUsecase) VerifyClient(ctx context.Context, clientID string) error {
	clientIDerr := r.AuthenticationsRepository.VerifyClientID(ctx, clientID)
	if(clientIDerr != nil) {
		return clientIDerr
	}

	return nil
}

func (r *AuthenticationsUsecase) VerifySecretToken(ctx context.Context, secretToken string) error {
	secretTokenErr := r.AuthenticationsRepository.VerifySecretToken(ctx, secretToken)
	if(secretTokenErr != nil) {
		return secretTokenErr
	}

	return nil
}

func (r *AuthenticationsUsecase) VerifyAccessToken(ctx context.Context, bearerToken string) (*models.AccessTokenClaims, error ){
	authToken, authTokenErr := r.AuthenticationsRepository.VerifyAuthToken(ctx, bearerToken)
	if(authTokenErr != nil) {
		return nil, authTokenErr
	}

	userClaim, claimErr := r.TokenManagerSecurity.VerifyAccessToken(authToken, false)
	if(claimErr != nil) {
		return nil, claimErr
	}

	return userClaim, nil
}

func (r *AuthenticationsUsecase) VerifyRefreshToken(ctx context.Context, bearerToken string) (*models.RefreshTokenClaims, error ){
	authToken, authTokenErr := r.AuthenticationsRepository.VerifyAuthToken(ctx, bearerToken)
	if(authTokenErr != nil) {
		return nil, authTokenErr
	}

	userClaim, claimErr := r.TokenManagerSecurity.VerifyRefreshToken(authToken, false)
	if(claimErr != nil) {
		return nil, claimErr
	}

	return userClaim, nil
}
