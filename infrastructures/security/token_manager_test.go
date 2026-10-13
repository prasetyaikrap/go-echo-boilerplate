package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"
)

func newTestTokenManager(accessSecret, refreshSecret string) *TokenManagerSecurity {
	configs := &configurations.Configs{
		Envs: &models.ENVConfig{Application: models.ApplicationConfig{
			ClientID:         "TEST",
			JWTAccessSecret:  accessSecret,
			JWTRefreshSecret: refreshSecret,
		}},
		Logger: configurations.NewSlogLogger(),
	}

	return NewTokenManagerSecurity(configs)
}

func TestAccessTokenRoundTrip(t *testing.T) {
	tokenManager := newTestTokenManager("access-secret", "refresh-secret")

	token, err := tokenManager.GenerateAccessToken(nil, "web", "session-1", nil, time.Minute)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := tokenManager.VerifyAccessToken(token, false)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.SessionID != "session-1" || claims.Issuer != "TEST" {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestRefreshTokenRoundTrip(t *testing.T) {
	tokenManager := newTestTokenManager("access-secret", "refresh-secret")

	token, err := tokenManager.GenerateRefreshToken(nil, "web", "session-2", nil, time.Minute)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := tokenManager.VerifyRefreshToken(token, false)
	if err != nil || claims.SessionID != "session-2" {
		t.Errorf("verify: got (%+v, %v)", claims, err)
	}

	if _, err := tokenManager.VerifyAccessToken(token, false); err == nil {
		t.Error("a refresh token must not verify as an access token")
	}
}

func TestVerifyAccessTokenRejects(t *testing.T) {
	tokenManager := newTestTokenManager("access-secret", "refresh-secret")
	claims := models.AccessTokenClaims{SessionID: "x"}

	otherSecret, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("other-secret"))
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte("access-secret"))
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	expired, _ := tokenManager.GenerateAccessToken(nil, "web", "x", nil, -time.Minute)

	tests := map[string]string{
		"signed with another secret": otherSecret,
		"HS512 with the same secret": hs512,
		"alg none":                   none,
		"expired":                    expired,
		"garbage":                    "not-a-token",
	}

	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := tokenManager.VerifyAccessToken(token, false); err == nil {
				t.Error("token should be rejected")
			}
		})
	}
}

func TestVerifyAccessTokenIgnoreExpiry(t *testing.T) {
	tokenManager := newTestTokenManager("access-secret", "refresh-secret")
	expired, _ := tokenManager.GenerateAccessToken(nil, "web", "x", nil, -time.Minute)

	if _, err := tokenManager.VerifyAccessToken(expired, true); err != nil {
		t.Errorf("ignoreExpiry should accept an expired token: %v", err)
	}
}

func TestTokensWithoutSecret(t *testing.T) {
	tokenManager := newTestTokenManager("", "")

	if _, err := tokenManager.GenerateAccessToken(nil, "web", "s", nil, time.Minute); err == nil {
		t.Error("generating an access token without a secret must fail")
	}
	if _, err := tokenManager.GenerateRefreshToken(nil, "web", "s", nil, time.Minute); err == nil {
		t.Error("generating a refresh token without a secret must fail")
	}

	forged, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, models.AccessTokenClaims{SessionID: "forged"}).SignedString([]byte(""))
	if _, err := tokenManager.VerifyAccessToken(forged, false); err == nil {
		t.Error("a token signed with an empty key must be rejected")
	}
}
