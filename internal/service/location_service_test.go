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

type stubLocationDB struct {
	err   error
	saved *domain.Location
}

func (s *stubLocationDB) SaveCheckHistory(location *domain.Location, ctx context.Context, timeOut time.Duration) error {
	s.saved = location
	return s.err
}

type stubLocationCache struct {
	resp []*domain.Incident
	err  error
}

func (s *stubLocationCache) Check(location *domain.Location, ctx context.Context, timeOut time.Duration) ([]*domain.Incident, error) {
	return s.resp, s.err
}

type stubLocationQueue struct {
	enqueued bool
	err      error
	task     domain.LocationCheckTask
}

func (s *stubLocationQueue) EnqueueLocationCheck(taskData domain.LocationCheckTask, ctx context.Context, timeOut time.Duration) error {
	s.enqueued = true
	s.task = taskData
	return s.err
}

func testLocationConfig() *config.App {
	return &config.App{DB: config.DB{TimeOuts: config.TimeOuts{Read: time.Second, Write: time.Second}}, Logger: config.Logger{BuffSize: 10, Mode: "dev", Level: "debug"}}
}

func testLocLogger(cfg *config.App) *logger.Service {
	return logger.NewService(cfg)
}

func TestLocation_InvalidUID(t *testing.T) {
	cfg := testLocationConfig()
	svc := NewLocation(&stubLocationDB{}, &stubLocationCache{}, nil, testLocLogger(cfg), cfg)
	_, err := svc.Check(0, 0, "not-uuid", context.Background())
	if err == nil {
		t.Fatal("expected error for invalid uuid")
	}
}

func TestLocation_Check_InDangerEnqueued(t *testing.T) {
	cfg := testLocationConfig()
	inc := domain.NewIncident("t", "d", 0, 0, 100, domain.SeverityLow, "type")
	cache := &stubLocationCache{resp: []*domain.Incident{inc}}
	queue := &stubLocationQueue{}
	db := &stubLocationDB{}
	svc := NewLocation(db, cache, queue, testLocLogger(cfg), cfg)

	uid := uuid.New().String()
	incidents, err := svc.Check(inc.Latitude, inc.Longitude, uid, context.Background())
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

func TestLocation_Check_CacheError(t *testing.T) {
	cfg := testLocationConfig()
	cache := &stubLocationCache{err: errors.New("cache down")}
	svc := NewLocation(&stubLocationDB{}, cache, nil, testLocLogger(cfg), cfg)
	_, err := svc.Check(0, 0, uuid.New().String(), context.Background())
	if err == nil {
		t.Fatal("expected error from cache")
	}
}

func TestLocation_Check_NoIncidents_NoEnqueue(t *testing.T) {
	cfg := testLocationConfig()
	cache := &stubLocationCache{resp: []*domain.Incident{}}
	queue := &stubLocationQueue{}
	db := &stubLocationDB{}
	svc := NewLocation(db, cache, queue, testLocLogger(cfg), cfg)

	_, err := svc.Check(0, 0, uuid.New().String(), context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queue.enqueued {
		t.Fatal("expected no enqueue when no incidents")
	}
}

func TestLocation_Check_SaveHistoryErrorNonFatal(t *testing.T) {
	cfg := testLocationConfig()
	cache := &stubLocationCache{resp: []*domain.Incident{}}
	db := &stubLocationDB{err: errors.New("db fail")}
	svc := NewLocation(db, cache, nil, testLocLogger(cfg), cfg)

	_, err := svc.Check(0, 0, uuid.New().String(), context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLocation_Check_NilQueueReturnsError(t *testing.T) {
	cfg := testLocationConfig()
	inc := domain.NewIncident("t", "d", 0, 0, 100, domain.SeverityLow, "type")
	cache := &stubLocationCache{resp: []*domain.Incident{inc}}
	svc := NewLocation(&stubLocationDB{}, cache, nil, testLocLogger(cfg), cfg)

	_, err := svc.Check(inc.Latitude, inc.Longitude, uuid.New().String(), context.Background())
	if err == nil {
		t.Fatalf("expected error when queue is nil")
	}
}
