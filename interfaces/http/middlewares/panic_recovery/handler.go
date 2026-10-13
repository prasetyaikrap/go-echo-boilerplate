package panic_recovery

import (
	"log/slog"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

// NewPanicRecoveryMiddleware turns a handler panic into a generic 500 and logs it with the stack trace and request_id
func NewPanicRecoveryMiddleware() echo.MiddlewareFunc {
	return echoMiddleware.RecoverWithConfig(echoMiddleware.RecoverConfig{
		DisableStackAll: true,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			slog.ErrorContext(c.Request().Context(), "Panic recovered",
				"method", c.Request().Method,
				"uri", c.Request().RequestURI,
				"error", err.Error(),
				"stack", string(stack),
			)

			return err
		},
	})
}
