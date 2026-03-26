package handlers

import (
	"context"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/web/dto"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SystemService defines the interface consumed by the system handler.
type SystemService interface {
	Health(ctx context.Context) domain.HealthStatus
}

// System is the HTTP handler for system endpoints.
type System struct {
	svc SystemService
}

// NewSystem creates a System handler.
func NewSystem(svc SystemService) *System {
	return &System{svc: svc}
}

// @Summary Статус системы
// @Description Возвращает состояние компонентов системы
// @Tags system
// @Produce json
// @Success 200 {object} dto.SystemHealthResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/system/health [get]
func (h *System) Health(c *gin.Context) {
	status := h.svc.Health(c.Request.Context())

	httpStatus := http.StatusOK
	svcStatus := "ok"

	if !status.Database || !status.Redis {
		httpStatus = http.StatusServiceUnavailable
		svcStatus = "degraded"
	}

	c.JSON(httpStatus, dto.SystemHealthResponse{
		Status: svcStatus,
		Checks: dto.HealthChecks{
			Database: status.Database,
			Redis:    status.Redis,
		},
	})
}
