package routers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"geoNotifications/internal/config"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/logger"
	"geoNotifications/internal/web/handlers"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutes_Basic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	cfg := &config.App{Logger: config.Logger{BuffSize: 10, Mode: "dev", Level: "debug"}}
	logSvc := logger.NewService(cfg)

	incSvc := incServiceStub{}
	locSvc := locServiceStub{}
	sysSvc := sysServiceStub{}

	incHandler := handlers.NewIncidents(incSvc)
	locHandler := handlers.NewLocation(locSvc)
	sysHandler := handlers.NewSystem(sysSvc)

	RegisterRoutes(eng, locHandler, incHandler, sysHandler, logSvc, "key")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/incidents/stats", nil)
	eng.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 stats, got %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/incidents", bytes.NewBufferString(`{}`))
	eng.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without key, got %d", w2.Code)
	}

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/api/v1/system/health", nil)
	eng.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 health, got %d", w3.Code)
	}
}

type incServiceStub struct{}

func (s incServiceStub) Create(string, string, float64, float64, float64, string, string, context.Context) (*domain.Incident, error) {
	return domain.NewIncident("t", "d", 0, 0, 1, domain.SeverityLow, "type"), nil
}
func (s incServiceStub) GetStats(ctx context.Context) (int, error) { return 1, nil }
func (s incServiceStub) GetAllIncidents(ctx context.Context, page, limit int) ([]domain.Incident, int, error) {
	return []domain.Incident{}, 0, nil
}
func (s incServiceStub) GetByID(ctx context.Context, id string) (*domain.Incident, error) {
	return domain.NewIncident("t", "d", 0, 0, 1, domain.SeverityLow, "type"), nil
}
func (s incServiceStub) Update(id, title, description string, latitude, longitude, radius float64, severity, incidentType string, ctx context.Context) (*domain.Incident, error) {
	return domain.NewIncident("t", "d", 0, 0, 1, domain.SeverityLow, "type"), nil
}
func (s incServiceStub) Delete(ctx context.Context, id string) error { return nil }

type locServiceStub struct{}

func (l locServiceStub) Check(latitude, longitude float64, uid string, ctx context.Context) ([]*domain.Incident, error) {
	return []*domain.Incident{}, nil
}

type sysServiceStub struct{}

func (s sysServiceStub) Health(ctx context.Context) domain.HealthStatus {
	return domain.HealthStatus{Database: true, Redis: true}
}
