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

type Postgres struct {
	db     *pgxpool.Pool
	logger *logger.Service
	cfg    *config.App
}

func NewPostgres(cfg *config.App, logger *logger.Service) (*Postgres, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%d/%s?sslmode=%s",
		cfg.DB.Postgres.Host,
		cfg.DB.Postgres.Port,
		cfg.DB.Postgres.DBName,
		cfg.DB.Postgres.SSLMode,
	)

	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse pg config: %w", err)
	}

	poolCfg.ConnConfig.Password = cfg.DB.Postgres.Password
	poolCfg.ConnConfig.User = cfg.DB.Postgres.User

	poolCfg.MaxConns = int32(cfg.DB.Postgres.MaxOpenConns)
	poolCfg.MaxConnLifetime = cfg.DB.Postgres.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pg pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	logger.Log(zap.InfoLevel, "connected to postgres")

	return &Postgres{
		db:     pool,
		logger: logger,
		cfg:    cfg,
	}, nil
}

func (p *Postgres) Close() {
	p.db.Close()
}

func (p *Postgres) Ping(ctx context.Context, timeOut time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()
	err := p.db.Ping(ctxWithTimeout)
	if err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}
