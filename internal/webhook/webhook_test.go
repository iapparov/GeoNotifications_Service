package webhook

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"geoNotifications/internal/config"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/logger"

	"github.com/google/uuid"
)

func newTestLogger() *logger.Service {
	cfg := &config.App{Logger: config.Logger{BuffSize: 1, Mode: "dev", Level: "debug"}}
	return logger.NewService(cfg)
}

func TestSend_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := NewSender(srv.URL, newTestLogger())

	task := domain.LocationCheckTask{
		Location: &domain.Location{
			ID:        uuid.New(),
			Latitude:  0,
			Longitude: 0,
			CreatedAt: time.Now(),
			InDanger:  false,
		},
		Incidents: []*domain.Incident{},
	}
	if err := sender.Send(task); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestSend_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	sender := NewSender(srv.URL, newTestLogger())

	task := domain.LocationCheckTask{
		Location: &domain.Location{
			ID:        uuid.New(),
			Latitude:  0,
			Longitude: 0,
			CreatedAt: time.Now(),
			InDanger:  false,
		},
		Incidents: []*domain.Incident{},
	}
	if err := sender.Send(task); err == nil {
		t.Fatal("expected error for bad status")
	}
}
