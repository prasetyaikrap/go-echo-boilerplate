package utils

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type SuccessResponseConfig struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    any 		`json:"data"`
}

type SuccessResponseWithMetadataConfig struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    any 		`json:"data"`
	Metadata Metadata	`json:"metadata"`
}

type SuccessResponseSSEventConfig struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type Metadata struct {
	TotalCount 	int64 `json:"total_count"`
	TotalPage  	int64 `json:"total_page"`
	CurrentPage int64 `json:"current_page"`
	PerPage    	int64 `json:"per_page"`
	NextCursor 	*string `json:"next_cursor,omitempty"`
	PrevCursor 	*string `json:"prev_cursor,omitempty"`
}

type ErrorResponseConfig struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Error   any 		`json:"error"`
}

func SuccessResponse(c echo.Context, response SuccessResponseConfig) error {
	if response.Code == 204 {
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(response.Code, response)
}

func ErrorResponse(c echo.Context, err error) error {
	if err == nil {
		return nil
	}

	errorResponse := buildErrorResponse(c, err)
	return c.JSON(errorResponse.Code, errorResponse)
}

// buildErrorResponse is the error body shared by JSON and SSE error responses
func buildErrorResponse(c echo.Context, err error) ErrorResponseConfig {
	errorID := uuid.New().String()

	// Handle Echo's built-in HTTPError (e.g. 404, 405, etc)
	if he, ok := err.(*echo.HTTPError); ok {
		return ErrorResponseConfig{
			Code:    he.Code,
			Message: http.StatusText(he.Code),
			Error: map[string]any{
				"id":   errorID,
				"type": "HTTP_ERROR",
				"code": he.Code,
				"data": he.Message,
			},
		}
	}

	// Handle custom application exceptions
	exceptions, ok := err.(*Exceptions)
	if !ok {
		return internalServerErrorResponse(c, err, errorID)
	}

	return ErrorResponseConfig{
		Code:    exceptions.Code,
		Message: exceptions.Error(),
		Error: map[string]any{
			"id":   errorID,
			"type": exceptions.Type,
			"code": exceptions.Code,
			"data": exceptions.ErrorObject,
		},
	}
}

func SuccessResponseWithMetadata(c echo.Context, response SuccessResponseWithMetadataConfig) error {
	if response.Code == 204 {
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(response.Code, response)
}

func HttpErrorHandler(err error, c echo.Context) {
	if c.Response().Committed { 
		return 
	}
	ErrorResponse(c, err)
}

// SuccessResponseSSEvent writes one Server-Sent Event. A string or []byte Data is sent as-is (so pre-encoded JSON
// is not encoded twice); any other value is JSON-encoded. Multi-line data is split into one "data:" line per line,
// which clients join back with "\n"
func SuccessResponseSSEvent(c echo.Context, response SuccessResponseSSEventConfig) error {
	data, err := encodeSSEData(response.Data)
	if err != nil {
		return err
	}

	return writeSSEvent(c, response.Event, data)
}

// ErrorResponseSSEvent writes an "error" event whose data is the same JSON body as ErrorResponse
func ErrorResponseSSEvent(c echo.Context, err error) error {
	if err == nil {
		return nil
	}

	payload, marshalErr := json.Marshal(buildErrorResponse(c, err))
	if marshalErr != nil {
		return marshalErr
	}

	return writeSSEvent(c, "error", string(payload))
}

func encodeSSEData(data any) (string, error) {
	switch value := data.(type) {
	case nil:
		return "", nil
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}
}

func writeSSEvent(c echo.Context, event string, data string) error {
	w := c.Response()
	if !w.Committed {
		w.Header().Set(echo.HeaderContentType, "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no") // disable nginx proxy buffering
		w.WriteHeader(http.StatusOK)
	}

	var frame strings.Builder
	// A newline in the event name would start a new field, so it is removed
	if event = strings.NewReplacer("\r", "", "\n", "").Replace(event); event != "" {
		frame.WriteString("event: " + event + "\n")
	}

	// A raw newline would end the data field, so every line gets its own "data:" prefix
	for _, line := range strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n") {
		frame.WriteString("data: " + strings.TrimSuffix(line, "\r") + "\n")
	}
	frame.WriteString("\n")

	if _, err := w.Write([]byte(frame.String())); err != nil {
		return err
	}
	w.Flush()
	return nil
}

// internalServerErrorResponse logs the real error server-side and returns a generic response, so internal details
// (SQL errors, hostnames, file paths) never reach the client. The error id in the response matches the log line
func internalServerErrorResponse(c echo.Context, err error, errorID string) ErrorResponseConfig {
	slog.ErrorContext(c.Request().Context(), "Internal server error",
		"error_id", errorID,
		"method", c.Request().Method,
		"uri", c.Request().RequestURI,
		"error", err.Error(),
	)

	return ErrorResponseConfig{
		Code:    http.StatusInternalServerError,
		Message: http.StatusText(http.StatusInternalServerError),
		Error: map[string]any{
			"id":   errorID,
			"type": "INTERNALSERVER_ERROR",
			"code": http.StatusInternalServerError,
			"data": nil,
		},
	}
}
