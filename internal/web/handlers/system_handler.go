package handlers

import (
	"context"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/web/dto"
	"net/http"

	"github.com/gin-gonic/gin"
)

type System struct {
	service SystemService
}

type SystemService interface {
	Health(ctx context.Context) domain.HealthStatus
}

func NewSystem(service SystemService) *System {
	return &System{service: service}
}

// @Summary Статус системы
// @Description Возвращает состояние компонентов системы
// @Tags system
// @Produce json
// @Success 200 {object} dto.SystemHealthResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/system/health [get]
func (h *System) Health(c *gin.Context) {
	status := h.service.Health(c.Request.Context())

	httpStatus := http.StatusOK
	serviceStatus := "ok"

	if !status.Database || !status.Redis {
		httpStatus = http.StatusServiceUnavailable
		serviceStatus = "degraded"
	}

	response := dto.SystemHealthResponse{
		Status: serviceStatus,
		Checks: dto.HealthChecks{
			Database: status.Database,
			Redis:    status.Redis,
		},
	}

	c.JSON(httpStatus, response)
}
