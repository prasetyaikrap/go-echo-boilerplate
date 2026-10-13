package logger

import (
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func NewSlogLoggerMiddleware(configs *configurations.Configs) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		Skipper: func(c echo.Context) bool {
			return c.Path() == models.HealthLivenessPath || c.Path() == models.HealthReadinessPath
		},
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogError:    true,
		HandleError: true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			// The request context carries the request_id, which the logger adds to the record
			ctx := c.Request().Context()

			if v.Error == nil {
				configs.Logger.LogAttrs(ctx, slog.LevelInfo, "REQUEST",
					slog.String("method", v.Method),
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.Duration("latency", v.Latency),
				)
			} else {
				configs.Logger.LogAttrs(ctx, slog.LevelError, "REQUEST_ERROR",
					slog.String("method", v.Method),
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.Duration("latency", v.Latency),
					slog.String("err", v.Error.Error()),
				)
			}
			return nil
		},
	})
}
