package handlers

import (
	"context"
	"errors"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/web/dto"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Incidents struct {
	service IncidentsService
}

type IncidentsService interface {
	Create(title, description string, latitude, longitude, radius float64, severity, incidentType string, ctx context.Context) (*domain.Incident, error)
	GetStats(ctx context.Context) (int, error)
	GetAllIncidents(ctx context.Context, page, limit int) ([]domain.Incident, int, error)
	GetByID(ctx context.Context, id string) (*domain.Incident, error)
	Update(id, title, description string, latitude, longitude, radius float64, severity, incidentType string, ctx context.Context) (*domain.Incident, error)
	Delete(ctx context.Context, id string) error
}

func NewIncidents(service IncidentsService) *Incidents {
	return &Incidents{service: service}
}

// @Summary Создать инцидент
// @Description Создает новый инцидент
// @Tags incidents
// @Accept json
// @Produce json
// @Param incident body dto.IncidentRequestCreate true "Данные инцидента"
// @Success 201 {object} domain.Incident
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/incidents [post]
func (i *Incidents) Create(c *gin.Context) {
	var inc dto.IncidentRequestCreate
	if err := c.ShouldBindJSON(&inc); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}
	incident, err := i.service.Create(inc.Title, inc.Description, inc.Latitude, inc.Longitude, inc.Radius, inc.Severity, inc.IncidentType, c.Request.Context())
	if i.incidentsValidation(err) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, incident)
}

// @Summary Статистика инцидентов
// @Description Возвращает общее количество инцидентов
// @Tags incidents
// @Produce json
// @Success 200 {object} map[string]int "total_incidents"
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/incidents/stats [get]
func (i *Incidents) GetStats(c *gin.Context) {
	count, err := i.service.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unique_user_id": count})
}

// @Summary Список инцидентов
// @Description Возвращает инциденты с пагинацией
// @Tags incidents
// @Produce json
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Размер страницы" default(10)
// @Success 200 {array} domain.Incident
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/incidents [get]
func (i *Incidents) GetAllIncidents(c *gin.Context) {
	// Pagination parameters
	page := c.Query("page")
	pInt, err := strconv.Atoi(page)
	if err != nil || pInt < 1 {
		pInt = 1
	}
	limit := c.Query("limit")
	lInt, err := strconv.Atoi(limit)
	if err != nil || lInt < 1 {
		lInt = 10
	}

	incidents, total, err := i.service.GetAllIncidents(c.Request.Context(), pInt, lInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"incidents": incidents,
	})

}

// @Summary Получить инцидент по ID
// @Description Возвращает инцидент по его идентификатору
// @Tags incidents
// @Produce json
// @Param id path string true "Incident ID"
// @Success 200 {object} domain.Incident
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/incidents/{id} [get]
func (i *Incidents) GetIncidentByID(c *gin.Context) {
	id := c.Param("id")
	incident, err := i.service.GetByID(c.Request.Context(), id)
	if errors.Is(err, domain.ErrInvalidID) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}
	if errors.Is(err, domain.ErrIncidentNotFound) {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Message: err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, incident)
}

// @Summary Обновить инцидент
// @Description Обновляет данные инцидента по ID
// @Tags incidents
// @Accept json
// @Produce json
// @Param id path string true "Incident ID"
// @Param incident body dto.IncidentRequestUpdate true "Новые данные инцидента"
// @Success 200 {object} domain.Incident
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/incidents/{id} [put]
func (i *Incidents) PutIncident(c *gin.Context) {
	id := c.Param("id")
	_, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "invalid incident ID"})
		return
	}
	var inc dto.IncidentRequestUpdate
	if err := c.ShouldBindJSON(&inc); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}
	incident, err := i.service.Update(id, inc.Title, inc.Description, inc.Latitude, inc.Longitude, inc.Radius, inc.Severity, inc.IncidentType, c.Request.Context())
	if i.incidentsValidation(err) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, incident)
}

// @Summary Удалить инцидент
// @Description Удаляет инцидент по ID
// @Tags incidents
// @Param id path string true "Incident ID"
// @Success 204
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/incidents/{id} [delete]
func (i *Incidents) DeleteIncident(c *gin.Context) {
	id := c.Param("id")

	err := i.service.Delete(c.Request.Context(), id)

	if errors.Is(err, domain.ErrInvalidID) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (i *Incidents) incidentsValidation(err error) bool {
	switch {
	case errors.Is(err, domain.ErrInvalidTitle),
		errors.Is(err, domain.ErrInvalidDescription),
		errors.Is(err, domain.ErrInvalidLatitude),
		errors.Is(err, domain.ErrInvalidLongitude),
		errors.Is(err, domain.ErrInvalidRadius),
		errors.Is(err, domain.ErrInvalidSeverity),
		errors.Is(err, domain.ErrInvalidType):
		return true
	default:
		return false
	}
}
