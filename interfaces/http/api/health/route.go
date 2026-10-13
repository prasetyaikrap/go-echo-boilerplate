package health

import (
	"go-serviceboilerplate/commons/models"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes registers the probes at the root (not under an API prefix), where Kubernetes expects them.
// Both paths are listed in models.ClientVerificationSkipPaths, so probes need no X-Client-Id header
func (h *HealthHandler) RegisterRoutes(e *echo.Echo) {
	e.GET(models.HealthLivenessPath, h.Liveness)
	e.GET(models.HealthReadinessPath, h.Readiness)
}
