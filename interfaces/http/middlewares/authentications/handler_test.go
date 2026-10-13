package authentications

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"go-serviceboilerplate/applications/usecases"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastructures/configurations"
	"go-serviceboilerplate/infrastructures/repositories"
	"go-serviceboilerplate/infrastructures/security"
)

func newTestMiddleware(t *testing.T) (*AuthMiddlewareHandler, *security.TokenManagerSecurity) {
	t.Helper()

	configs := &configurations.Configs{
		Envs: &models.ENVConfig{Application: models.ApplicationConfig{
			ClientID:         "TEST",
			AllowedCleintIDs: []string{"web"},
			SecretToken:      "service-token",
			JWTAccessSecret:  "access-secret",
			JWTRefreshSecret: "refresh-secret",
		}},
		Logger: configurations.NewSlogLogger(),
	}

	tokenManager := security.NewTokenManagerSecurity(configs)
	usecase := usecases.NewAuthenticationsUsecase(repositories.NewAuthenticationsRepositories(nil, configs), tokenManager, nil)

	return NewAuthMiddlewareHandler(usecase), tokenManager
}

func serve(e *echo.Echo, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestVerifyClient(t *testing.T) {
	middleware, _ := newTestMiddleware(t)

	e := echo.New()
	e.Use(middleware.VerifyClient)
	e.GET("/system", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET(models.HealthReadinessPath, func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	if rec := serve(e, "/system", map[string]string{models.XClientIdHeader: "web"}); rec.Code != http.StatusOK {
		t.Errorf("allowed client: status = %d, want 200", rec.Code)
	}
	if rec := serve(e, "/system", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("missing client id: status = %d, want 401", rec.Code)
	}
	if rec := serve(e, models.HealthReadinessPath, nil); rec.Code != http.StatusOK {
		t.Errorf("skip path: status = %d, want 200 without client id", rec.Code)
	}
}

func TestVerifyAuthenticationStoresClaims(t *testing.T) {
	middleware, tokenManager := newTestMiddleware(t)
	accessToken, _ := tokenManager.GenerateAccessToken(nil, "web", "access-session", nil, time.Minute)
	refreshToken, _ := tokenManager.GenerateRefreshToken(nil, "web", "refresh-session", nil, time.Minute)

	e := echo.New()
	e.GET("/me", func(c echo.Context) error {
		ctx := c.Request().Context()
		if claims, ok := utils.AccessTokenClaimsFromContext(ctx); ok {
			if c.Get(models.ContextUserClaim) == nil {
				t.Error("claims missing from echo.Context")
			}
			return c.String(http.StatusOK, "access:"+claims.SessionID)
		}
		if claims, ok := utils.RefreshTokenClaimsFromContext(ctx); ok {
			return c.String(http.StatusOK, "refresh:"+claims.SessionID)
		}
		return c.NoContent(http.StatusInternalServerError)
	}, middleware.VerifyAuthentication)

	rec := serve(e, "/me", map[string]string{models.AuthorizationHeader: "Bearer " + accessToken})
	if rec.Code != http.StatusOK || rec.Body.String() != "access:access-session" {
		t.Errorf("access token: got %d %q", rec.Code, rec.Body.String())
	}

	rec = serve(e, "/me", map[string]string{models.AuthorizationHeader: "Bearer " + refreshToken, models.XRenewTokenHeader: "true"})
	if rec.Code != http.StatusOK || rec.Body.String() != "refresh:refresh-session" {
		t.Errorf("renew with refresh token: got %d %q", rec.Code, rec.Body.String())
	}

	if rec := serve(e, "/me", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: status = %d, want 401", rec.Code)
	}
}

func TestVerifyAuthenticationWithServiceToken(t *testing.T) {
	middleware, tokenManager := newTestMiddleware(t)
	accessToken, _ := tokenManager.GenerateAccessToken(nil, "web", "s", nil, time.Minute)

	e := echo.New()
	e.GET("/internal", func(c echo.Context) error { return c.NoContent(http.StatusOK) }, middleware.VerifyAuthenticationWithServiceToken)

	bearer := "Bearer " + accessToken
	tests := []struct {
		name         string
		serviceToken string
		bearer       string
		want         int
	}{
		{name: "both valid", serviceToken: "service-token", bearer: bearer, want: http.StatusOK},
		{name: "wrong service token", serviceToken: "wrong", bearer: bearer, want: http.StatusUnauthorized},
		{name: "missing bearer", serviceToken: "service-token", bearer: "", want: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(e, "/internal", map[string]string{models.XServiceTokenHeader: tt.serviceToken, models.AuthorizationHeader: tt.bearer})
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
