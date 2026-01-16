package handlers

import (
	"context"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/web/dto"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Location struct {
	service LocationService
}

type LocationService interface {
	Check(latitude, longitude float64, uid string, ctx context.Context) ([]*domain.Incident, error)
}

func NewLocation(service LocationService) *Location {
	return &Location{service: service}
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
func (l *Location) Check(c *gin.Context) {
	var locationReq dto.LocationRequest
	if err := c.ShouldBindJSON(&locationReq); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}
	incidents, err := l.service.Check(locationReq.Latitude, locationReq.Longitude, locationReq.UID, c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"incidents": incidents})
}
