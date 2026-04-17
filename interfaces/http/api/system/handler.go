package system

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"go-serviceboilerplate/applications/usecases"
	"go-serviceboilerplate/commons/utils"
)

type SystemHandler struct {
	systemUsecase *usecases.SystemUsecase
}

func NewSystemHandler(systemUsecase *usecases.SystemUsecase ) *SystemHandler {
	return &SystemHandler{systemUsecase}
}

func (h *SystemHandler) GetSystemInfo(c echo.Context) error {
	systemInfo := h.systemUsecase.GetSystemInfo()

	return utils.SuccessResponse(c, utils.SuccessResponseConfig{
		Code:    http.StatusOK,
		Message: "System information retrieved successfully",
		Data:    systemInfo,
	})
}