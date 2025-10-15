package security

import (
	"errors"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastrucutres/configurations"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManagerSecurity struct {
	Configs 	*configurations.Configs
}

func NewTokenManagerSecurity(configs *configurations.Configs) *TokenManagerSecurity { 
	return &TokenManagerSecurity{configs}
}

func (s *TokenManagerSecurity) GenerateAccessToken(user any, client_id, session_id string, permissions []string, expiration time.Duration) (string, error) {
	envConfigs := s.Configs.Envs
	jwtSecret := []byte(envConfigs.Application.JWTAccessSecret)
	now := time.Now().UTC()
	claims := models.AccessTokenClaims{
		Profile: models.UserProfileClaims{},
		SessionID: session_id,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "userID",
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer: envConfigs.Application.ClientID,
			Audience: jwt.ClaimStrings{client_id},
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func (s *TokenManagerSecurity) GenerateRefreshToken(user any, client_id, session_id string, permissions []string, expiration time.Duration) (string, error) {
	envConfigs := s.Configs.Envs
	jwtSecret := []byte(envConfigs.Application.JWTRefreshSecret)
	now := time.Now().UTC()
	claims := models.RefreshTokenClaims{
		Profile: models.UserProfileClaims{},
		SessionID: session_id,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "userID",
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer: envConfigs.Application.ClientID,
			Audience: jwt.ClaimStrings{client_id},
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func (s *TokenManagerSecurity) VerifyAccessToken(tokenStr string, ignoreExpiry bool) (*models.AccessTokenClaims, error) {
	envConfigs := s.Configs.Envs
	jwtSecret := []byte(envConfigs.Application.JWTAccessSecret)
	opts := []jwt.ParserOption{}
	if(ignoreExpiry) {
		opts = append(opts, jwt.WithoutClaimsValidation())
	}
	token, err := jwt.ParseWithClaims(tokenStr, &models.AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	}, opts...)

	if err != nil {
		return nil, utils.NewAuthenticationError(err)
	}

	claims, ok := token.Claims.(*models.AccessTokenClaims)

	if(!token.Valid || !ok) {
		return nil, utils.NewAuthenticationError(errors.New("invalid access token"))
	}
	return claims, nil
}

func (s *TokenManagerSecurity) VerifyRefreshToken(tokenStr string, ignoreExpiry bool) (*models.RefreshTokenClaims, error) {
	envConfigs := s.Configs.Envs
	jwtSecret := []byte(envConfigs.Application.JWTRefreshSecret)
	opts := []jwt.ParserOption{}
	if(ignoreExpiry) {
		opts = append(opts, jwt.WithoutClaimsValidation())
	}

	token, err := jwt.ParseWithClaims(tokenStr, &models.RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	}, opts...)

	if err != nil {
		return nil, utils.NewAuthenticationError(err)
	}

	claims, ok := token.Claims.(*models.RefreshTokenClaims)

	if(!token.Valid || !ok) {
		return nil, utils.NewAuthenticationError(errors.New("invalid refresh token"))
	}
	return claims, nil
}

