package request_context

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastructures/configurations"
)

func TestRequestIDMiddleware(t *testing.T) {
	e := echo.New()
	e.Use(NewRequestIDMiddleware())
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, utils.RequestIDFromContext(c.Request().Context()))
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	generated := rec.Header().Get(models.XRequestIDHeader)
	if generated == "" || rec.Body.String() != generated {
		t.Errorf("generated id: header %q, ctx %q", generated, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(models.XRequestIDHeader, "trace-123")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Header().Get(models.XRequestIDHeader) != "trace-123" || rec.Body.String() != "trace-123" {
		t.Errorf("incoming id: header %q, ctx %q", rec.Header().Get(models.XRequestIDHeader), rec.Body.String())
	}
}

func TestRequestTimeoutMiddleware(t *testing.T) {
	tests := []struct {
		name         string
		timeout      time.Duration
		wantDeadline bool
	}{
		{name: "sets a deadline", timeout: time.Second, wantDeadline: true},
		{name: "zero disables it", timeout: 0, wantDeadline: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs := &configurations.Configs{
				Envs:   &models.ENVConfig{Application: models.ApplicationConfig{RequestTimeout: tt.timeout}},
				Logger: configurations.NewSlogLogger(),
			}

			var hasDeadline bool
			e := echo.New()
			e.Use(NewRequestTimeoutMiddleware(configs))
			e.GET("/", func(c echo.Context) error {
				_, hasDeadline = c.Request().Context().Deadline()
				return c.NoContent(http.StatusOK)
			})

			e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

			if hasDeadline != tt.wantDeadline {
				t.Errorf("deadline set = %v, want %v", hasDeadline, tt.wantDeadline)
			}
		})
	}
}

func TestRequestTimeoutSkipsStreamingRoutes(t *testing.T) {
	original := models.RequestTimeoutSkipPaths
	models.RequestTimeoutSkipPaths = []string{"/agents/:id/chat"}
	t.Cleanup(func() { models.RequestTimeoutSkipPaths = original })

	configs := &configurations.Configs{
		Envs:   &models.ENVConfig{Application: models.ApplicationConfig{RequestTimeout: time.Second}},
		Logger: configurations.NewSlogLogger(),
	}

	deadlines := map[string]bool{}
	e := echo.New()
	e.Use(NewRequestTimeoutMiddleware(configs))
	handler := func(c echo.Context) error {
		_, deadlines[c.Path()] = c.Request().Context().Deadline()
		return c.NoContent(http.StatusOK)
	}
	e.POST("/agents/:id/chat", handler)
	e.POST("/agents/:id", handler)

	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/agents/1/chat", nil))
	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/agents/1", nil))

	if deadlines["/agents/:id/chat"] {
		t.Error("streaming route should have no deadline")
	}
	if !deadlines["/agents/:id"] {
		t.Error("other routes should keep the deadline")
	}
}
