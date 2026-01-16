package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"geoNotifications/internal/config"
)

type stubDBChecker struct{ err error }

func (s *stubDBChecker) Ping(ctx context.Context, d time.Duration) error { return s.err }

type stubRedisChecker struct{ err error }

func (s *stubRedisChecker) Ping(ctx context.Context, d time.Duration) error { return s.err }

func TestSystemHealth_AllOK(t *testing.T) {
	cfg := &config.App{DB: config.DB{TimeOuts: config.TimeOuts{Read: time.Second}}}
	svc := NewSystem(&stubDBChecker{}, &stubRedisChecker{}, cfg)

	status := svc.Health(context.Background())
	if !status.Database || !status.Redis {
		t.Fatalf("expected both healthy, got %+v", status)
	}
}

func TestSystemHealth_Degraded(t *testing.T) {
	cfg := &config.App{DB: config.DB{TimeOuts: config.TimeOuts{Read: time.Second}}}
	svc := NewSystem(&stubDBChecker{err: errors.New("db down")}, &stubRedisChecker{err: errors.New("redis down")}, cfg)

	status := svc.Health(context.Background())
	if status.Database || status.Redis {
		t.Fatalf("expected degraded, got %+v", status)
	}
}
