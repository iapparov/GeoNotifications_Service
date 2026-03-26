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

type stubIncidentsDB struct {
	createCalled bool
	updateCalled bool
	deleteCalled bool
	gotIncident  *domain.Incident
	getByIDResp  *domain.Incident
	getByIDErr   error
	err          error
}

func (s *stubIncidentsDB) Create(ctx context.Context, incident *domain.Incident, timeout time.Duration) error {
	s.createCalled = true
	s.gotIncident = incident
	return s.err
}
func (s *stubIncidentsDB) Update(ctx context.Context, incident *domain.Incident, timeout time.Duration) error {
	s.updateCalled = true
	s.gotIncident = incident
	return s.err
}
func (s *stubIncidentsDB) GetStats(ctx context.Context, timeout time.Duration) (int, error) {
	return 0, s.err
}
func (s *stubIncidentsDB) GetAllIncidents(ctx context.Context, page, limit int, timeout time.Duration) ([]domain.Incident, int, error) {
	return nil, 0, s.err
}
func (s *stubIncidentsDB) GetByID(ctx context.Context, id uuid.UUID, timeout time.Duration) (*domain.Incident, error) {
	return s.getByIDResp, s.getByIDErr
}
func (s *stubIncidentsDB) Delete(ctx context.Context, id uuid.UUID, timeout time.Duration) error {
	s.deleteCalled = true
	return s.err
}

type stubIncidentsCache struct {
	setErr       error
	getResp      *domain.Incident
	getErr       error
	deleteErr    error
	setCalled    bool
	deleteCalled bool
}

func (s *stubIncidentsCache) Set(ctx context.Context, incident *domain.Incident, timeout time.Duration) error {
	s.setCalled = true
	return s.setErr
}
func (s *stubIncidentsCache) Delete(ctx context.Context, id uuid.UUID, timeout time.Duration) error {
	s.deleteCalled = true
	return s.deleteErr
}
func (s *stubIncidentsCache) GetByID(ctx context.Context, id uuid.UUID, timeout time.Duration) (*domain.Incident, error) {
	return s.getResp, s.getErr
}

type stubCacheRetryQueue struct{ enqueued bool }

func (s *stubCacheRetryQueue) Enqueue(incident *domain.Incident) { s.enqueued = true }

// --- helpers ---

func testIncidentsConfig() *config.App {
	return &config.App{
		Incident: config.Incident{TitleMinLength: 2, TitleMaxLength: 100, DescriptionMinLength: 2, DescriptionMaxLength: 200},
		Retry:    config.Retry{MaxAttempts: 1, Delay: 0, Backoff: 1},
		DB:       config.DB{TimeOuts: config.TimeOuts{Write: time.Second, Read: time.Second}},
		Logger:   config.Logger{BuffSize: 10, Mode: "dev", Level: "debug"},
	}
}

func testLogger(cfg *config.App) *logger.Service {
	return logger.NewService(cfg)
}

// --- tests ---

func TestIncidents_Create_InvalidTitle(t *testing.T) {
	cfg := testIncidentsConfig()
	svc := NewIncidents(&stubIncidentsDB{}, &stubIncidentsCache{}, &stubCacheRetryQueue{}, testLogger(cfg), cfg)

	_, err := svc.Create(context.Background(), "a", "desc", 0, 0, 1, string(domain.SeverityLow), "type")
	if !errors.Is(err, domain.ErrInvalidTitle) {
		t.Fatalf("expected ErrInvalidTitle, got %v", err)
	}
}

func TestIncidents_Create_CacheErrorEnqueueRetry(t *testing.T) {
	cfg := testIncidentsConfig()
	db := &stubIncidentsDB{}
	cache := &stubIncidentsCache{setErr: errors.New("cache down")}
	retryQueue := &stubCacheRetryQueue{}
	svc := NewIncidents(db, cache, retryQueue, testLogger(cfg), cfg)

	inc, err := svc.Create(context.Background(), "title", "description", 10, 10, 100, string(domain.SeverityLow), "type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !db.createCalled {
		t.Fatal("db.Create not called")
	}
	if !cache.setCalled {
		t.Fatal("cache.Set not called")
	}
	if !retryQueue.enqueued {
		t.Fatal("cache retry enqueue not triggered")
	}
	if inc == nil || inc.Title != "title" {
		t.Fatalf("unexpected incident: %+v", inc)
	}
}

func TestIncidents_Create_DbError(t *testing.T) {
	cfg := testIncidentsConfig()
	db := &stubIncidentsDB{err: errors.New("db down")}
	svc := NewIncidents(db, &stubIncidentsCache{}, &stubCacheRetryQueue{}, testLogger(cfg), cfg)

	_, err := svc.Create(context.Background(), "title", "description", 1, 1, 1, string(domain.SeverityLow), "type")
	if err == nil {
		t.Fatal("expected db error")
	}
}

func TestIncidents_GetByID_CacheHit(t *testing.T) {
	cfg := testIncidentsConfig()
	expected := domain.NewIncident("t", "d", 0, 0, 1, domain.SeverityLow, "type")
	cache := &stubIncidentsCache{getResp: expected}
	svc := NewIncidents(&stubIncidentsDB{}, cache, &stubCacheRetryQueue{}, testLogger(cfg), cfg)

	got, err := svc.GetByID(context.Background(), expected.ID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != expected.ID {
		t.Fatalf("expected %s, got %s", expected.ID, got.ID)
	}
}

func TestIncidents_GetByID_CacheMissFallsBackToDB(t *testing.T) {
	cfg := testIncidentsConfig()
	expected := domain.NewIncident("t", "d", 0, 0, 1, domain.SeverityLow, "type")
	db := &stubIncidentsDB{getByIDResp: expected}
	cache := &stubIncidentsCache{getErr: domain.ErrIncidentNotFound}
	svc := NewIncidents(db, cache, &stubCacheRetryQueue{}, testLogger(cfg), cfg)

	got, err := svc.GetByID(context.Background(), expected.ID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != expected.ID {
		t.Fatalf("expected %s, got %s", expected.ID, got.ID)
	}
}

func TestIncidents_GetByID_InvalidID(t *testing.T) {
	cfg := testIncidentsConfig()
	svc := NewIncidents(&stubIncidentsDB{}, &stubIncidentsCache{}, &stubCacheRetryQueue{}, testLogger(cfg), cfg)

	_, err := svc.GetByID(context.Background(), "not-a-uuid")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestIncidents_Update_CacheErrorEnqueue(t *testing.T) {
	cfg := testIncidentsConfig()
	existing := domain.NewIncident("t", "d", 0, 0, 1, domain.SeverityLow, "type")
	db := &stubIncidentsDB{getByIDResp: existing}
	cache := &stubIncidentsCache{setErr: errors.New("cache down"), getErr: domain.ErrIncidentNotFound}
	retryQueue := &stubCacheRetryQueue{}
	svc := NewIncidents(db, cache, retryQueue, testLogger(cfg), cfg)

	_, err := svc.Update(context.Background(), existing.ID.String(), "t2", "d2", 1, 1, 2, string(domain.SeverityHigh), "type2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !retryQueue.enqueued {
		t.Fatal("expected enqueue on cache error")
	}
}

func TestIncidents_Update_DbError(t *testing.T) {
	cfg := testIncidentsConfig()
	existing := domain.NewIncident("t", "d", 0, 0, 1, domain.SeverityLow, "type")
	db := &stubIncidentsDB{getByIDResp: existing, err: errors.New("db down")}
	cache := &stubIncidentsCache{getErr: domain.ErrIncidentNotFound}
	svc := NewIncidents(db, cache, &stubCacheRetryQueue{}, testLogger(cfg), cfg)

	_, err := svc.Update(context.Background(), existing.ID.String(), "t2", "d2", 1, 1, 2, string(domain.SeverityHigh), "type2")
	if err == nil {
		t.Fatal("expected db error")
	}
	if !db.updateCalled {
		t.Fatal("expected db update to be called")
	}
}

func TestIncidents_Delete_CacheError(t *testing.T) {
	cfg := testIncidentsConfig()
	cache := &stubIncidentsCache{deleteErr: errors.New("cache fail")}
	svc := NewIncidents(&stubIncidentsDB{}, cache, &stubCacheRetryQueue{}, testLogger(cfg), cfg)

	err := svc.Delete(context.Background(), uuid.New().String())
	if err == nil {
		t.Fatal("expected cache error to bubble up")
	}
	if !cache.deleteCalled {
		t.Fatal("expected cache delete called")
	}
}

func TestIncidents_Create_InvalidDescription(t *testing.T) {
	cfg := testIncidentsConfig()
	svc := NewIncidents(&stubIncidentsDB{}, &stubIncidentsCache{}, &stubCacheRetryQueue{}, testLogger(cfg), cfg)

	_, err := svc.Create(context.Background(), "tt", "d", 0, 0, 1, string(domain.SeverityLow), "type")
	if !errors.Is(err, domain.ErrInvalidDescription) {
		t.Fatalf("expected ErrInvalidDescription, got %v", err)
	}
}

func TestIncidents_GetStats_Error(t *testing.T) {
	cfg := testIncidentsConfig()
	svc := NewIncidents(&stubIncidentsDB{err: errors.New("db fail")}, &stubIncidentsCache{}, &stubCacheRetryQueue{}, testLogger(cfg), cfg)

	_, err := svc.GetStats(context.Background())
	if err == nil {
		t.Fatal("expected error from db")
	}
}

func TestIncidents_GetAllIncidents_Error(t *testing.T) {
	cfg := testIncidentsConfig()
	svc := NewIncidents(&stubIncidentsDB{err: errors.New("db fail")}, &stubIncidentsCache{}, &stubCacheRetryQueue{}, testLogger(cfg), cfg)

	_, _, err := svc.GetAllIncidents(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected error from db")
	}
}
