package utils

import (
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
	Data    []any 		`json:"data"`
	Metadata Metadata	`json:"metadata"`
}

type Metadata struct {
	TotalCount 	int `json:"total_count"`
	TotalPage  	int `json:"total_page"`
	CurrentPage int `json:"current_page"`
	PerPage    	int `json:"per_page"`
}

type ErrorResponseConfig struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Error   any 		`json:"error"`
}

func SuccessResponse(c echo.Context, response SuccessResponseConfig) error {
	return c.JSON(response.Code, response)
}

func ErrorResponse(c echo.Context, response ErrorResponseConfig) error {
	return c.JSON(response.Code, response)
}

func SuccessResponseWithMetadata(c echo.Context, response SuccessResponseWithMetadataConfig) error {
	return c.JSON(response.Code, response)
}