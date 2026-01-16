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

type Redis struct {
	client *redis.Client
	logger *logger.Service
	cfg    *config.App
	db     DB
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

type DB interface {
	WarmUp(ctx context.Context, timeout time.Duration) ([]*domain.Incident, error)
}

func NewRedis(logger *logger.Service, cfg *config.App, db DB) (*Redis, error) {
	addr := fmt.Sprintf("%s:%d", cfg.DB.Redis.Host, cfg.DB.Redis.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.DB.Redis.Password,
		DB:       cfg.DB.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	logger.Log(zap.InfoLevel, "connected to redis")

	return &Redis{
		client: client,
		logger: logger,
		cfg:    cfg,
		db:     db,
	}, nil
}

func (r *Redis) Close() error {
	if err := r.client.Close(); err != nil {
		return fmt.Errorf("closing redis client: %w", err)
	}
	return nil
}

func (r *Redis) Ping(ctx context.Context, timeOut time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	if err := r.client.Ping(ctxWithTimeout).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}
