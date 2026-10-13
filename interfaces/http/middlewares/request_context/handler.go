package request_context

import (
	"slices"

	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastructures/configurations"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

// NewRequestIDMiddleware reuses the incoming X-Request-Id header (so IDs propagate across services) or generates one,
// returns it in the response header, and stores it in the request context so every log line carries it.
// Register it first so the ID is available to all other middlewares.
func NewRequestIDMiddleware() echo.MiddlewareFunc {
	return echoMiddleware.RequestIDWithConfig(echoMiddleware.RequestIDConfig{
		RequestIDHandler: func(c echo.Context, requestID string) {
			utils.SetRequestContext(c, utils.WithRequestID(c.Request().Context(), requestID))
		},
	})
}

// NewRequestTimeoutMiddleware sets a deadline on the request context (REQUEST_TIMEOUT). Work that receives the ctx
// (database, cache, broker, outgoing HTTP) is cancelled when it passes, and the request returns 503 Service Unavailable.
// Routes in models.RequestTimeoutSkipPaths (streaming) get no deadline.
func NewRequestTimeoutMiddleware(configs *configurations.Configs) echo.MiddlewareFunc {
	timeout := configs.Envs.Application.RequestTimeout
	if timeout <= 0 {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}

	return echoMiddleware.ContextTimeoutWithConfig(echoMiddleware.ContextTimeoutConfig{
		Timeout: timeout,
		Skipper: func(c echo.Context) bool {
			return slices.Contains(models.RequestTimeoutSkipPaths, c.Path())
		},
	})
}
