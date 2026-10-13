package utils_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"go-serviceboilerplate/commons/utils"
)

func runErrorResponse(t *testing.T, err error) (int, map[string]any) {
	t.Helper()

	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)

	if responseErr := utils.ErrorResponse(c, err); responseErr != nil {
		t.Fatalf("ErrorResponse returned %v", responseErr)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body %q: %v", rec.Body.String(), err)
	}

	return rec.Code, body
}

func TestErrorResponseApplicationException(t *testing.T) {
	code, body := runErrorResponse(t, utils.NewNotFoundError(errors.New("user not found")))

	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
	if body["message"] != "user not found" {
		t.Errorf("message = %v, want the exception message", body["message"])
	}
	if errorBody := body["error"].(map[string]any); errorBody["type"] != utils.NotFoundError {
		t.Errorf("type = %v, want %s", errorBody["type"], utils.NotFoundError)
	}
}

func TestErrorResponseHidesInternalErrors(t *testing.T) {
	code, body := runErrorResponse(t, errors.New(`pq: relation "users_secret_table" does not exist at db-prod-01:5432`))

	if code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", code)
	}
	if body["message"] != http.StatusText(http.StatusInternalServerError) {
		t.Errorf("message = %v, want a generic message", body["message"])
	}

	raw, _ := json.Marshal(body)
	if strings.Contains(string(raw), "users_secret_table") || strings.Contains(string(raw), "db-prod-01") {
		t.Errorf("response leaks internal details: %s", raw)
	}
	if id, _ := body["error"].(map[string]any)["id"].(string); id == "" {
		t.Error("response must include an error id")
	}
}

func TestErrorResponseEchoHTTPError(t *testing.T) {
	code, body := runErrorResponse(t, echo.NewHTTPError(http.StatusMethodNotAllowed))

	if code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", code)
	}
	if errorBody := body["error"].(map[string]any); errorBody["type"] != "HTTP_ERROR" {
		t.Errorf("type = %v, want HTTP_ERROR", errorBody["type"])
	}
}
