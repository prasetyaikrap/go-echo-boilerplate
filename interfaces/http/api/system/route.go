package system

import (
	"github.com/labstack/echo/v4"
)

func (h *SystemHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.GetSystemInfo)
}