package panic_recovery

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"go-serviceboilerplate/commons/utils"
)

func TestPanicRecoveryMiddleware(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = utils.HttpErrorHandler
	e.Use(NewPanicRecoveryMiddleware())
	e.GET("/", func(c echo.Context) error {
		panic("secret internal state")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret internal state") {
		t.Errorf("response leaks the panic value: %s", rec.Body.String())
	}
}
