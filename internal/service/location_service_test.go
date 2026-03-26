package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"geoNotifications/internal/config"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/logger"

	"github.com/google/uuid"
)

// --- stubs ---

type stubLocationDB struct {
	err        error
	saved      *domain.Location
	nearbyResp []*domain.Incident
	nearbyErr  error
}

func (s *stubLocationDB) SaveCheckHistory(ctx context.Context, loc *domain.Location, timeout time.Duration) error {
	s.saved = loc
	return s.err
}

func (s *stubLocationDB) FindNearby(ctx context.Context, lat, lon float64, timeout time.Duration) ([]*domain.Incident, error) {
	return s.nearbyResp, s.nearbyErr
}

type stubLocationQueue struct {
	enqueued bool
	err      error
	task     domain.LocationCheckTask
}

func (s *stubLocationQueue) EnqueueLocationCheck(ctx context.Context, task domain.LocationCheckTask, timeout time.Duration) error {
	s.enqueued = true
	s.task = task
	return s.err
}

// --- helpers ---

func testLocationConfig() *config.App {
	return &config.App{
		DB:     config.DB{TimeOuts: config.TimeOuts{Read: time.Second, Write: time.Second}},
		Logger: config.Logger{BuffSize: 10, Mode: "dev", Level: "debug"},
	}
}

func testLocLogger(cfg *config.App) *logger.Service {
	return logger.NewService(cfg)
}

// --- tests ---

func TestLocation_InvalidUID(t *testing.T) {
	cfg := testLocationConfig()
	svc := NewLocation(&stubLocationDB{}, nil, testLocLogger(cfg), cfg)

	_, err := svc.Check(context.Background(), 0, 0, "not-uuid")
	if err == nil {
		t.Fatal("expected error for invalid uuid")
	}
}

func TestLocation_Check_InDangerEnqueued(t *testing.T) {
	cfg := testLocationConfig()
	inc := domain.NewIncident("t", "d", 0, 0, 100, domain.SeverityLow, "type")
	db := &stubLocationDB{nearbyResp: []*domain.Incident{inc}}
	queue := &stubLocationQueue{}
	svc := NewLocation(db, queue, testLocLogger(cfg), cfg)

	uid := uuid.New().String()
	incidents, err := svc.Check(context.Background(), inc.Latitude, inc.Longitude, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(incidents))
	}
	if !queue.enqueued {
		t.Fatal("expected enqueue")
	}
	if db.saved == nil || !db.saved.InDanger {
		t.Fatal("expected saved location marked InDanger")
	}
}

func TestLocation_Check_DBError(t *testing.T) {
	cfg := testLocationConfig()
	db := &stubLocationDB{nearbyErr: errors.New("db down")}
	svc := NewLocation(db, nil, testLocLogger(cfg), cfg)

	_, err := svc.Check(context.Background(), 0, 0, uuid.New().String())
	if err == nil {
		t.Fatal("expected error from db")
	}
}

func TestLocation_Check_NoIncidents_NoEnqueue(t *testing.T) {
	cfg := testLocationConfig()
	db := &stubLocationDB{nearbyResp: []*domain.Incident{}}
	queue := &stubLocationQueue{}
	svc := NewLocation(db, queue, testLocLogger(cfg), cfg)

	_, err := svc.Check(context.Background(), 0, 0, uuid.New().String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queue.enqueued {
		t.Fatal("expected no enqueue when no incidents")
	}
}

func TestLocation_Check_SaveHistoryErrorNonFatal(t *testing.T) {
	cfg := testLocationConfig()
	db := &stubLocationDB{err: errors.New("db fail"), nearbyResp: []*domain.Incident{}}
	svc := NewLocation(db, nil, testLocLogger(cfg), cfg)

	_, err := svc.Check(context.Background(), 0, 0, uuid.New().String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLocation_Check_NilQueueReturnsError(t *testing.T) {
	cfg := testLocationConfig()
	inc := domain.NewIncident("t", "d", 0, 0, 100, domain.SeverityLow, "type")
	db := &stubLocationDB{nearbyResp: []*domain.Incident{inc}}
	svc := NewLocation(db, nil, testLocLogger(cfg), cfg)

	_, err := svc.Check(context.Background(), inc.Latitude, inc.Longitude, uuid.New().String())
	if err == nil {
		t.Fatal("expected error when queue is nil")
	}
}
