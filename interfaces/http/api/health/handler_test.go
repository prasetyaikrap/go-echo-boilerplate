package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"
)

func serveHealth(t *testing.T, checks map[string]ReadinessCheck, path string) *httptest.ResponseRecorder {
	t.Helper()

	configs := &configurations.Configs{Envs: &models.ENVConfig{}, Logger: configurations.NewSlogLogger()}
	e := echo.New()
	NewHealthHandler(configs, checks).RegisterRoutes(e)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestLivenessIgnoresDependencies(t *testing.T) {
	failing := map[string]ReadinessCheck{"database": func(context.Context) error { return errors.New("down") }}

	if rec := serveHealth(t, failing, models.HealthLivenessPath); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 even when a dependency is down", rec.Code)
	}
}

func TestReadiness(t *testing.T) {
	ok := func(context.Context) error { return nil }
	down := func(context.Context) error { return errors.New("dial tcp db-prod-01:5432: connection refused") }

	rec := serveHealth(t, map[string]ReadinessCheck{"database": ok, "redis": ok}, models.HealthReadinessPath)
	if rec.Code != http.StatusOK {
		t.Errorf("all checks ok: status = %d, want 200", rec.Code)
	}

	rec = serveHealth(t, map[string]ReadinessCheck{"database": ok, "redis": down}, models.HealthReadinessPath)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("one check down: status = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"redis":"unavailable"`) {
		t.Errorf("body should report the failing check: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "db-prod-01") {
		t.Errorf("body leaks the error detail: %s", rec.Body.String())
	}
}
