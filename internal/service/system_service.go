package service

import (
	"context"
	"geoNotifications/internal/config"
	"geoNotifications/internal/domain"
	"time"
)

type System struct {
	db     DBChecker
	redis  RedisChecker
	config *config.App
}

type DBChecker interface {
	Ping(ctx context.Context, timeOut time.Duration) error
}

type RedisChecker interface {
	Ping(ctx context.Context, timeOut time.Duration) error
}

func NewSystem(db DBChecker, redis RedisChecker, config *config.App) *System {
	return &System{db: db, redis: redis, config: config}
}

func (s *System) Health(ctx context.Context) domain.HealthStatus {
	status := domain.HealthStatus{
		Database: true,
		Redis:    true,
	}

	if err := s.db.Ping(ctx, s.config.DB.TimeOuts.Read); err != nil {
		status.Database = false
	}

	if err := s.redis.Ping(ctx, s.config.DB.TimeOuts.Read); err != nil {
		status.Redis = false
	}

	return status
}
