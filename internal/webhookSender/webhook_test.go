package webhookSender

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

func TestSendWebhook_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.App{WebHook: srv.URL, Logger: config.Logger{BuffSize: 1, Mode: "dev", Level: "debug"}}
	logSvc := logger.NewService(cfg)
	sender := NewWebhookSender(cfg, logSvc)

	task := domain.LocationCheckTask{Location: &domain.Location{
		ID:        uuid.New(),
		Latitude:  0,
		Longitude: 0,
		CreatedAT: time.Now(),
		InDanger:  false},
		Incidents: []*domain.Incident{}}
	if err := sender.SendWebhook(task); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestSendWebhook_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	cfg := &config.App{WebHook: srv.URL, Logger: config.Logger{BuffSize: 1, Mode: "dev", Level: "debug"}}
	logSvc := logger.NewService(cfg)
	sender := NewWebhookSender(cfg, logSvc)

	task := domain.LocationCheckTask{Location: &domain.Location{
		ID:        uuid.New(),
		Latitude:  0,
		Longitude: 0,
		CreatedAT: time.Now(),
		InDanger:  false},
		Incidents: []*domain.Incident{}}
	if err := sender.SendWebhook(task); err == nil {
		t.Fatal("expected error for bad status")
	}
}
