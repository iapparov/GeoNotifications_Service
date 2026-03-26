package handlers

import (
	"context"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/web/dto"
	"net/http"

	"github.com/gin-gonic/gin"
)

// LocationService defines the interface consumed by the location handler.
type LocationService interface {
	Check(ctx context.Context, lat, lon float64, uid string) ([]*domain.Incident, error)
}

// Location is the HTTP handler for location endpoints.
type Location struct {
	svc LocationService
}

// NewLocation creates a Location handler.
func NewLocation(svc LocationService) *Location {
	return &Location{svc: svc}
}

// @Summary Проверить местоположение
// @Description Проверяет местоположение пользователя на наличие опасных инцидентов
// @Tags location
// @Accept json
// @Produce json
// @Param location body dto.LocationRequest true "Координаты пользователя"
// @Success 200 {array} domain.Incident
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/location/check [post]
func (h *Location) Check(c *gin.Context) {
	var req dto.LocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}

	incidents, err := h.svc.Check(c.Request.Context(), req.Latitude, req.Longitude, req.UID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"incidents": incidents})
}
