package redis

import (
	"context"
	"fmt"
	"geoNotifications/internal/config"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/logger"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// Redis wraps a redis client and provides cache and queue operations.
type Redis struct {
	client *redis.Client
	log    *logger.Service
	cfg    *config.App
	db     DB
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// DB is used to load data for cache warm-up.
type DB interface {
	WarmUp(ctx context.Context, timeout time.Duration) ([]*domain.Incident, error)
}

// NewRedis creates a Redis client and verifies connectivity.
func NewRedis(log *logger.Service, cfg *config.App, db DB) (*Redis, error) {
	addr := fmt.Sprintf("%s:%d", cfg.DB.Redis.Host, cfg.DB.Redis.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.DB.Redis.Password,
		DB:       cfg.DB.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	log.Log(zap.InfoLevel, "connected to redis")

	return &Redis{
		client: client,
		log:    log,
		cfg:    cfg,
		db:     db,
	}, nil
}

// Close releases the redis client.
func (r *Redis) Close() error {
	if err := r.client.Close(); err != nil {
		return fmt.Errorf("close redis: %w", err)
	}
	return nil
}

// Ping checks redis connectivity within the given timeout.
func (r *Redis) Ping(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		return err
	}
	return nil
}
