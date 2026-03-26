package di

import (
	"context"
	"errors"
	"fmt"
	"geoNotifications/internal/config"
	"geoNotifications/internal/logger"
	"geoNotifications/internal/service"
	"geoNotifications/internal/storage/postgres"
	"geoNotifications/internal/storage/redis"
	"geoNotifications/internal/web/handlers"
	"geoNotifications/internal/web/routers"
	"geoNotifications/internal/webhook"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Container holds all application dependencies wired at startup.
type Container struct {
	Config      *config.App
	Logger      *logger.Service
	Postgres    *postgres.Postgres
	Redis       *redis.Redis
	CacheRetryQ *redis.CacheRetryQueue
	Webhook     *webhook.Sender
	IncidentSvc *service.Incidents
	LocationSvc *service.Location
	SystemSvc   *service.System
	Server      *http.Server
}

// Build constructs the full dependency graph.
func Build() (*Container, error) {
	cfg, err := config.NewAppConfig()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	log := logger.NewService(cfg)

	pg, err := postgres.NewPostgres(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	rdb, err := redis.NewRedis(log, cfg, pg)
	if err != nil {
		pg.Close()
		return nil, fmt.Errorf("redis: %w", err)
	}

	cacheRetryQ := redis.NewCacheRetryQueue(cfg)
	wh := webhook.NewSender(cfg.WebHook, log)

	incidentSvc := service.NewIncidents(pg, rdb, cacheRetryQ, log, cfg)
	locationSvc := service.NewLocation(pg, rdb, log, cfg)
	systemSvc := service.NewSystem(pg, rdb, cfg)

	incidentH := handlers.NewIncidents(incidentSvc)
	locationH := handlers.NewLocation(locationSvc)
	systemH := handlers.NewSystem(systemSvc)

	gin.SetMode(cfg.Gin.Mode)
	router := gin.New()
	routers.RegisterRoutes(router, locationH, incidentH, systemH, log, cfg.Auth.ApiKey)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	return &Container{
		Config:      cfg,
		Logger:      log,
		Postgres:    pg,
		Redis:       rdb,
		CacheRetryQ: cacheRetryQ,
		Webhook:     wh,
		IncidentSvc: incidentSvc,
		LocationSvc: locationSvc,
		SystemSvc:   systemSvc,
		Server:      srv,
	}, nil
}

// Start launches all background goroutines and the HTTP server.
func (c *Container) Start(ctx context.Context) error {
	c.Logger.StartLogger(ctx)
	c.Logger.Log(zapcore.InfoLevel, "logger started")

	c.Logger.Log(zapcore.InfoLevel, "warming up redis cache")
	if err := c.Redis.Ping(ctx, c.Config.DB.TimeOuts.Read); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	if err := c.Redis.WarmUp(ctx, c.Config.DB.TimeOuts.Read); err != nil {
		return fmt.Errorf("redis warmup: %w", err)
	}
	c.Logger.Log(zapcore.InfoLevel, "redis cache warmed up")

	c.Logger.Log(zapcore.InfoLevel, "starting webhook queue worker")
	c.Redis.StartQueue(ctx, c.Webhook)

	c.Logger.Log(zapcore.InfoLevel, "starting cache retry worker")
	c.CacheRetryQ.StartRetryWorker(ctx, c.Redis)

	c.Logger.Log(zapcore.InfoLevel, "starting HTTP server", zap.String("addr", c.Server.Addr))
	go func() {
		if err := c.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			c.Logger.Log(zapcore.FatalLevel, "http server failed", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully shuts down all components in reverse order.
func (c *Container) Stop(ctx context.Context) {
	c.Logger.Log(zapcore.InfoLevel, "shutting down HTTP server")
	if err := c.Server.Shutdown(ctx); err != nil {
		c.Logger.Log(zapcore.ErrorLevel, "http server shutdown", zap.Error(err))
	}

	c.Logger.Log(zapcore.InfoLevel, "stopping webhook worker")
	if err := c.Redis.StopQueue(ctx); err != nil {
		c.Logger.Log(zapcore.ErrorLevel, "webhook worker stop", zap.Error(err))
	}

	c.Logger.Log(zapcore.InfoLevel, "stopping cache retry worker")
	if err := c.CacheRetryQ.StopRetryWorker(ctx); err != nil {
		c.Logger.Log(zapcore.ErrorLevel, "cache retry stop", zap.Error(err))
	}

	c.Logger.Log(zapcore.InfoLevel, "closing redis")
	if err := c.Redis.Close(); err != nil {
		c.Logger.Log(zapcore.ErrorLevel, "redis close", zap.Error(err))
	}

	c.Logger.Log(zapcore.InfoLevel, "closing postgres")
	c.Postgres.Close()

	c.Logger.Log(zapcore.InfoLevel, "stopping logger")
	if err := c.Logger.Stop(ctx); err != nil {
		fmt.Println("logger stop error:", err)
	}
}
