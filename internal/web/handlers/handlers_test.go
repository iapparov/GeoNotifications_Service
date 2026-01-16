package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"geoNotifications/internal/domain"
	"geoNotifications/internal/web/middlewares"

	"github.com/gin-gonic/gin"
)

type incidentsServiceStub struct {
	createResp *domain.Incident
	createErr  error
	stats      int
	statsErr   error
	list       []domain.Incident
	total      int
	listErr    error
	getResp    *domain.Incident
	getErr     error
	updateResp *domain.Incident
	updateErr  error
	deleteErr  error
}

func (s *incidentsServiceStub) Create(title, description string, latitude, longitude, radius float64, severity, incidentType string, ctx context.Context) (*domain.Incident, error) {
	return s.createResp, s.createErr
}
func (s *incidentsServiceStub) GetStats(ctx context.Context) (int, error) { return s.stats, s.statsErr }
func (s *incidentsServiceStub) GetAllIncidents(ctx context.Context, page, limit int) ([]domain.Incident, int, error) {
	return s.list, s.total, s.listErr
}
func (s *incidentsServiceStub) GetByID(ctx context.Context, id string) (*domain.Incident, error) {
	return s.getResp, s.getErr
}
func (s *incidentsServiceStub) Update(id, title, description string, latitude, longitude, radius float64, severity, incidentType string, ctx context.Context) (*domain.Incident, error) {
	return s.updateResp, s.updateErr
}
func (s *incidentsServiceStub) Delete(ctx context.Context, id string) error { return s.deleteErr }

type locationServiceStub struct {
	resp []*domain.Incident
	err  error
}

func (s *locationServiceStub) Check(latitude, longitude float64, uid string, ctx context.Context) ([]*domain.Incident, error) {
	return s.resp, s.err
}

type systemServiceStub struct{ status domain.HealthStatus }

func (s *systemServiceStub) Health(ctx context.Context) domain.HealthStatus { return s.status }

func init() {
	gin.SetMode(gin.TestMode)
}

func performRequest(r http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestIncidents_Create_ValidationError(t *testing.T) {
	svc := &incidentsServiceStub{createErr: domain.ErrInvalidTitle}
	h := NewIncidents(svc)
	r := gin.New()
	r.POST("/incidents", h.Create)

	body := []byte(`{"description":"d","latitude":0,"longitude":0,"radius":1,"severity":"low","incident_type":"type"}`)
	w := performRequest(r, http.MethodPost, "/incidents", body, map[string]string{"Content-Type": "application/json"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestIncidents_Create_Success(t *testing.T) {
	inc := domain.NewIncident("t", "d", 10, 20, 1, domain.SeverityLow, "type")
	svc := &incidentsServiceStub{createResp: inc}
	h := NewIncidents(svc)
	r := gin.New()
	r.POST("/incidents", h.Create)

	payload := map[string]any{
		"title": "t", "description": "d", "latitude": 10, "longitude": 20, "radius": 1, "severity": "low", "incident_type": "type",
	}
	buf, _ := json.Marshal(payload)
	w := performRequest(r, http.MethodPost, "/incidents", buf, map[string]string{"Content-Type": "application/json"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestIncidents_GetByID_NotFound(t *testing.T) {
	svc := &incidentsServiceStub{getErr: domain.ErrIncidentNotFound}
	h := NewIncidents(svc)
	r := gin.New()
	r.GET("/incidents/:id", h.GetIncidentByID)

	w := performRequest(r, http.MethodGet, "/incidents/123", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestIncidents_GetByID_InvalidID(t *testing.T) {
	svc := &incidentsServiceStub{getErr: domain.ErrInvalidID}
	h := NewIncidents(svc)
	r := gin.New()
	r.GET("/incidents/:id", h.GetIncidentByID)

	w := performRequest(r, http.MethodGet, "/incidents/invalid", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestIncidents_Delete_InvalidID(t *testing.T) {
	svc := &incidentsServiceStub{deleteErr: domain.ErrInvalidID}
	h := NewIncidents(svc)
	r := gin.New()
	r.DELETE("/incidents/:id", h.DeleteIncident)

	w := performRequest(r, http.MethodDelete, "/incidents/bad", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestIncidents_GetStats(t *testing.T) {
	svc := &incidentsServiceStub{stats: 5}
	h := NewIncidents(svc)
	r := gin.New()
	r.GET("/incidents/stats", h.GetStats)

	w := performRequest(r, http.MethodGet, "/incidents/stats", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLocation_Check_Success(t *testing.T) {
	svc := &locationServiceStub{resp: []*domain.Incident{}}
	h := NewLocation(svc)
	r := gin.New()
	r.POST("/location/check", h.Check)

	body := []byte(`{"latitude":10,"longitude":20,"uid":"00000000-0000-0000-0000-000000000001"}`)
	w := performRequest(r, http.MethodPost, "/location/check", body, map[string]string{"Content-Type": "application/json"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLocation_Check_BadRequest(t *testing.T) {
	svc := &locationServiceStub{}
	h := NewLocation(svc)
	r := gin.New()
	r.POST("/location/check", h.Check)

	w := performRequest(r, http.MethodPost, "/location/check", []byte(`{"latitude":10}`), map[string]string{"Content-Type": "application/json"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSystem_Health_Degraded(t *testing.T) {
	svc := &systemServiceStub{status: domain.HealthStatus{Database: false, Redis: true}}
	h := NewSystem(svc)
	r := gin.New()
	r.GET("/health", h.Health)

	w := performRequest(r, http.MethodGet, "/health", nil, nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(middlewares.AuthMiddleware("secret"))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	w1 := performRequest(r, http.MethodGet, "/ping", nil, nil)
	if w1.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without key, got %d", w1.Code)
	}

	w2 := performRequest(r, http.MethodGet, "/ping", nil, map[string]string{"X-API-Key": "wrong"})
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong key, got %d", w2.Code)
	}

	w3 := performRequest(r, http.MethodGet, "/ping", nil, map[string]string{"X-API-Key": "secret"})
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid key, got %d", w3.Code)
	}
}
