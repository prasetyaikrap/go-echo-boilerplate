package health

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastructures/configurations"
)

type ReadinessCheck func(ctx context.Context) error

type HealthHandler struct {
	configs *configurations.Configs
	checks  map[string]ReadinessCheck
}

func NewHealthHandler(configs *configurations.Configs, checks map[string]ReadinessCheck) *HealthHandler {
	return &HealthHandler{configs: configs, checks: checks}
}

// Liveness only reports that the process can serve HTTP. It must not check dependencies: a database outage would
// otherwise make Kubernetes restart every pod, which does not fix the database
func (h *HealthHandler) Liveness(c echo.Context) error {
	return utils.SuccessResponse(c, utils.SuccessResponseConfig{
		Code:    http.StatusOK,
		Message: "Service is alive",
		Data:    map[string]string{"status": "ok"},
	})
}

// Readiness runs every check. If any fails, it returns 503 so Kubernetes stops routing traffic to this pod until it
// recovers. Failure details are logged, not returned, since the endpoint is public
func (h *HealthHandler) Readiness(c echo.Context) error {
	ctx := c.Request().Context()

	results := make(map[string]string, len(h.checks))
	ready := true

	for name, check := range h.checks {
		if err := check(ctx); err != nil {
			h.configs.Logger.WarnContext(ctx, "Readiness check failed", "check", name, "error", err)
			results[name] = "unavailable"
			ready = false
			continue
		}

		results[name] = "ok"
	}

	if !ready {
		return utils.SuccessResponse(c, utils.SuccessResponseConfig{
			Code:    http.StatusServiceUnavailable,
			Message: "Service is not ready",
			Data:    map[string]any{"status": "unavailable", "checks": results},
		})
	}

	return utils.SuccessResponse(c, utils.SuccessResponseConfig{
		Code:    http.StatusOK,
		Message: "Service is ready",
		Data:    map[string]any{"status": "ok", "checks": results},
	})
}
