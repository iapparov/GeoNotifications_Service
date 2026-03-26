package postgres

import (
	"context"
	"fmt"
	"geoNotifications/internal/config"
	"geoNotifications/internal/logger"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Postgres wraps a pgx connection pool.
type Postgres struct {
	pool *pgxpool.Pool
	log  *logger.Service
	cfg  *config.App
}

// NewPostgres opens a connection pool and verifies connectivity.
func NewPostgres(cfg *config.App, log *logger.Service) (*Postgres, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.Postgres.User,
		cfg.DB.Postgres.Password,
		cfg.DB.Postgres.Host,
		cfg.DB.Postgres.Port,
		cfg.DB.Postgres.DBName,
		cfg.DB.Postgres.SSLMode,
	)

	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse pg config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.DB.Postgres.MaxOpenConns)
	poolCfg.MaxConnLifetime = cfg.DB.Postgres.ConnMaxLifetime

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pg pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	log.Log(zap.InfoLevel, "connected to postgres")

	return &Postgres{
		pool: pool,
		log:  log,
		cfg:  cfg,
	}, nil
}

// Close releases all pool resources.
func (p *Postgres) Close() {
	p.pool.Close()
}

// Ping checks database connectivity within the given timeout.
func (p *Postgres) Ping(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}
