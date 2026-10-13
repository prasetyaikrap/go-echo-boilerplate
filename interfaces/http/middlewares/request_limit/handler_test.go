package request_limit

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastructures/configurations"
)

func TestBodyLimitMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		limit   string
		size    int
		chunked bool
		want    int
	}{
		{name: "under the limit", limit: "1K", size: 512, want: http.StatusOK},
		{name: "over the limit", limit: "1K", size: 2048, want: http.StatusRequestEntityTooLarge},
		{name: "over the limit, chunked", limit: "1K", size: 2048, chunked: true, want: http.StatusRequestEntityTooLarge},
		{name: "zero disables it", limit: "0", size: 2048, want: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs := &configurations.Configs{
				Envs:   &models.ENVConfig{Application: models.ApplicationConfig{RequestBodyLimit: tt.limit}},
				Logger: configurations.NewSlogLogger(),
			}

			e := echo.New()
			e.HTTPErrorHandler = utils.HttpErrorHandler
			e.Use(NewBodyLimitMiddleware(configs))
			e.POST("/", func(c echo.Context) error {
				if _, err := io.Copy(io.Discard, c.Request().Body); err != nil {
					return err
				}
				return c.NoContent(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(make([]byte, tt.size)))
			if tt.chunked {
				req.ContentLength = -1
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
