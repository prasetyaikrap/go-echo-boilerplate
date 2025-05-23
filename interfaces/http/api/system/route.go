package system

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

func (h *SystemHandler) RegisterRoute(e *echo.Echo) {
	g := e.Group("/system")
	g.GET("", h.GetSystemInfo)

	fmt.Println("System route registered")
}