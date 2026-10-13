package request_limit

import (
	"go-serviceboilerplate/infrastructures/configurations"
	"strings"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/bytes"
)

// NewBodyLimitMiddleware rejects request bodies larger than REQUEST_BODY_LIMIT with 413 Request Entity Too Large.
// It checks both the Content-Length header and the bytes actually read, so chunked uploads cannot bypass it.
// An invalid limit stops startup; "0" disables the limit.
func NewBodyLimitMiddleware(configs *configurations.Configs) echo.MiddlewareFunc {
	limit := strings.TrimSpace(configs.Envs.Application.RequestBodyLimit)

	size, err := bytes.Parse(limit)
	if err != nil {
		configs.Logger.Fatal("Invalid REQUEST_BODY_LIMIT, expected a size such as 10M, 512K or 1G", "value", limit, "error", err)
	}

	if size == 0 {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}

	return echoMiddleware.BodyLimit(limit)
}
