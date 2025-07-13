package system

import (
	"fmt"
	"go-serviceboilerplate/interfaces/http/middlewares"

	"github.com/labstack/echo/v4"
)

func (h *SystemHandler) RegisterRoute(e *echo.Echo, middlewares *middlewares.AppMiddlewaresHandler) {
	g := e.Group("/system")
	g.GET("", h.GetSystemInfo)

	fmt.Println("System route registered")
}