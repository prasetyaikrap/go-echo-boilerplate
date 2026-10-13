package security

import (
	"errors"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastructures/configurations"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManagerSecurity struct {
	Configs 	*configurations.Configs
}

// JWT secrets are optional: only services that issue or verify user tokens need them.
// Without a secret, signing and verifying that token type is refused, so tokens can never be signed with an empty key
func NewTokenManagerSecurity(configs *configurations.Configs) *TokenManagerSecurity {
	if configs.Envs.Application.JWTAccessSecret == "" {
		configs.Logger.Warn("JWT_ACCESS_SECRET is not set: access tokens cannot be issued or verified, so authenticated routes will reject every request")
	}
	if configs.Envs.Application.JWTRefreshSecret == "" {
		configs.Logger.Warn("JWT_REFRESH_SECRET is not set: refresh tokens cannot be issued or verified")
	}

	return &TokenManagerSecurity{configs}
}

// errJWTNotConfigured is returned when a token operation needs a secret that is not set
var errJWTNotConfigured = errors.New("token authentication is not configured")

// parserOptions only accepts HS256 tokens, so a token cannot choose a different algorithm (algorithm confusion)
func parserOptions(ignoreExpiry bool) []jwt.ParserOption {
	opts := []jwt.ParserOption{jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})}
	if ignoreExpiry {
		opts = append(opts, jwt.WithoutClaimsValidation())
	}

	return opts
}

func (s *TokenManagerSecurity) GenerateAccessToken(user any, client_id, session_id string, permissions []string, expiration time.Duration) (string, error) {
	envConfigs := s.Configs.Envs
	if envConfigs.Application.JWTAccessSecret == "" {
		return "", errJWTNotConfigured
	}
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
	if envConfigs.Application.JWTRefreshSecret == "" {
		return "", errJWTNotConfigured
	}
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
	if envConfigs.Application.JWTAccessSecret == "" {
		return nil, utils.NewAuthenticationError(errJWTNotConfigured)
	}
	jwtSecret := []byte(envConfigs.Application.JWTAccessSecret)
	token, err := jwt.ParseWithClaims(tokenStr, &models.AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	}, parserOptions(ignoreExpiry)...)

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
	if envConfigs.Application.JWTRefreshSecret == "" {
		return nil, utils.NewAuthenticationError(errJWTNotConfigured)
	}
	jwtSecret := []byte(envConfigs.Application.JWTRefreshSecret)
	token, err := jwt.ParseWithClaims(tokenStr, &models.RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	}, parserOptions(ignoreExpiry)...)

	if err != nil {
		return nil, utils.NewAuthenticationError(err)
	}

	claims, ok := token.Claims.(*models.RefreshTokenClaims)

	if(!token.Valid || !ok) {
		return nil, utils.NewAuthenticationError(errors.New("invalid refresh token"))
	}
	return claims, nil
}

