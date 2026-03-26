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

// IncidentsService defines the interface consumed by the incidents handler.
type IncidentsService interface {
	Create(ctx context.Context, title, description string, lat, lon, radius float64, severity, incidentType string) (*domain.Incident, error)
	GetStats(ctx context.Context) (int, error)
	GetAllIncidents(ctx context.Context, page, limit int) ([]domain.Incident, int, error)
	GetByID(ctx context.Context, id string) (*domain.Incident, error)
	Update(ctx context.Context, id, title, description string, lat, lon, radius float64, severity, incidentType string) (*domain.Incident, error)
	Delete(ctx context.Context, id string) error
}

// Incidents is the HTTP handler for incident endpoints.
type Incidents struct {
	svc IncidentsService
}

// NewIncidents creates an Incidents handler.
func NewIncidents(svc IncidentsService) *Incidents {
	return &Incidents{svc: svc}
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
func (h *Incidents) Create(c *gin.Context) {
	var req dto.IncidentRequestCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}

	inc, err := h.svc.Create(c.Request.Context(),
		req.Title, req.Description,
		req.Latitude, req.Longitude, req.Radius,
		req.Severity, req.IncidentType,
	)
	if isValidationErr(err) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, inc)
}

// @Summary Статистика инцидентов
// @Description Возвращает общее количество инцидентов
// @Tags incidents
// @Produce json
// @Success 200 {object} map[string]int "total_incidents"
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/incidents/stats [get]
func (h *Incidents) GetStats(c *gin.Context) {
	count, err := h.svc.GetStats(c.Request.Context())
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
func (h *Incidents) GetAllIncidents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 {
		limit = 10
	}

	incidents, total, err := h.svc.GetAllIncidents(c.Request.Context(), page, limit)
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
func (h *Incidents) GetIncidentByID(c *gin.Context) {
	inc, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
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
	c.JSON(http.StatusOK, inc)
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
func (h *Incidents) PutIncident(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "invalid incident ID"})
		return
	}

	var req dto.IncidentRequestUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}

	inc, err := h.svc.Update(c.Request.Context(), id,
		req.Title, req.Description,
		req.Latitude, req.Longitude, req.Radius,
		req.Severity, req.IncidentType,
	)
	if isValidationErr(err) {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, inc)
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
func (h *Incidents) DeleteIncident(c *gin.Context) {
	err := h.svc.Delete(c.Request.Context(), c.Param("id"))
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

// isValidationErr returns true if err is a known domain validation error.
func isValidationErr(err error) bool {
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
