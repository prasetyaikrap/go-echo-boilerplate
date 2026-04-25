package utils

import (
	"net/http"

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
	return c.JSON(response.Code, response)
}

func ErrorResponse(c echo.Context, err error) error {
	errorID := uuid.New().String()
	if err == nil {
		return nil
	}

	// Handle Echo's built-in HTTPError (e.g. 404, 405, etc)
	if he, ok := err.(*echo.HTTPError); ok {
		errorResponse := ErrorResponseConfig{
			Code:    he.Code,
			Message: http.StatusText(he.Code),
			Error: map[string]any{
				"id": errorID,
				"type": "HTTP_ERROR",
				"code": he.Code,
				"data": he.Message,
			},
		}
		return c.JSON(he.Code, errorResponse)
	}

	// Handle custom application exceptions
	exceptions, ok := err.(*Exceptions);
	if !ok {
		errorResponse := ErrorResponseConfig{
			Code: http.StatusInternalServerError,
			Message: err.Error(),
			Error: map[string]any{
				"id": errorID,
				"type": "INTERNALSERVER_ERROR",
				"code": http.StatusInternalServerError,
				"data": nil,
			},
		}
		return c.JSON(http.StatusInternalServerError, errorResponse)
	}

	exceptionResponse := ErrorResponseConfig{
		Code: exceptions.Code,
		Message: exceptions.Error(),
		Error: map[string]any{
			"id": errorID,
			"type": exceptions.Type,
			"code": exceptions.Code,
			"data": exceptions.ErrorObject,
		},
	}

	return c.JSON(exceptions.Code, exceptionResponse)
}

func SuccessResponseWithMetadata(c echo.Context, response SuccessResponseWithMetadataConfig) error {
	return c.JSON(response.Code, response)
}

func HttpErrorHandler(err error, c echo.Context) {
	if c.Response().Committed { 
		return 
	}
	ErrorResponse(c, err)
}