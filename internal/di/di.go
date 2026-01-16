package di

import (
	"context"
	"errors"
	"fmt"
	"geoNotifications/internal/config"
	"geoNotifications/internal/logger"
	"geoNotifications/internal/storage/postgres"
	"geoNotifications/internal/storage/redis"
	"geoNotifications/internal/web/handlers"
	"geoNotifications/internal/web/routers"
	"geoNotifications/internal/webhookSender"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func StartHttpServer(lc fx.Lifecycle, hIncidents *handlers.Incidents, hSystem *handlers.System, hLocation *handlers.Location, config *config.App, logger *logger.Service) {

	gin.SetMode(config.Gin.Mode)
	router := gin.New()

	routers.RegisterRoutes(router, hLocation, hIncidents, hSystem, logger, config.Auth.ApiKey)

	address := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	server := &http.Server{
		Addr:    address,
		Handler: router,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Log(zapcore.InfoLevel, "Starting HTTP server", zap.String("address", address))
			go func() {
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Log(zap.FatalLevel, "http server failed", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Log(zapcore.InfoLevel, "Server stopped")
			return server.Shutdown(ctx)
		},
	})
}

func ClosePostgresOnStop(lc fx.Lifecycle, postgres *postgres.Postgres, logger *logger.Service) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Log(zapcore.InfoLevel, "Stopping postgres")
			postgres.Close()
			logger.Log(zapcore.InfoLevel, "Postgres stopped")
			return nil
		},
	})
}

func WarmUpRedisOnStart(lc fx.Lifecycle, redis *redis.Redis, logger *logger.Service, config *config.App) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Log(zapcore.InfoLevel, "Warming up Redis connection")
			err := redis.Ping(ctx, config.DB.TimeOuts.Read)
			if err != nil {
				logger.Log(zapcore.FatalLevel, "Error warming up Redis", zap.Error(err))
				return err
			}
			err = redis.WarmUp(ctx, config.DB.TimeOuts.Read)
			if err != nil {
				logger.Log(zapcore.FatalLevel, "Error warming up Redis", zap.Error(err))
				return err
			}
			logger.Log(zapcore.InfoLevel, "Redis connection warmed up")
			return nil
		},
	})
}

func CloseRedisOnStop(lc fx.Lifecycle, redis *redis.Redis, logger *logger.Service) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Log(zapcore.InfoLevel, "Stopping Redis")
			err := redis.Close()
			if err != nil {
				logger.Log(zapcore.ErrorLevel, "Error closing Redis", zap.Error(err))
				return err
			}
			logger.Log(zapcore.InfoLevel, "Redis stopped")
			return nil
		},
	})
}

func StartWebhookService(lc fx.Lifecycle, webhook *webhookSender.WebhookSender, redis *redis.Redis, logger *logger.Service) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Log(zapcore.InfoLevel, "Starting webhook service")
			redis.StartQueue(ctx, webhook)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Log(zapcore.InfoLevel, "Stopping webhook worker")
			err := redis.StopQueue(ctx)
			if err != nil {
				logger.Log(zapcore.ErrorLevel, "Error stopping webhook worker", zap.Error(err))
				return err
			}
			logger.Log(zapcore.InfoLevel, "Webhook service stopped")
			return nil
		},
	})
}

func StartCacheRetryService(lc fx.Lifecycle, retryQueue *redis.CacheRetryQueue, redis *redis.Redis, logger *logger.Service) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Log(zapcore.InfoLevel, "Starting cache retry service")
			retryQueue.StartRetryWorker(ctx, redis)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Log(zapcore.InfoLevel, "Stopping cache rertry service stopped")
			err := retryQueue.StopRetryWorker(ctx)
			if err != nil {
				logger.Log(zapcore.ErrorLevel, "Error stopping cache retry service", zap.Error(err))
				return err
			}
			logger.Log(zapcore.InfoLevel, "Cache retry service stopped")
			return nil
		},
	})
}

func StartLoggerAsync(lc fx.Lifecycle, logger *logger.Service) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Log(zapcore.InfoLevel, "Starting logger service")
			logger.StartLogger(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Log(zapcore.InfoLevel, "Logger service stopping...")
			err := logger.Stop(ctx)
			if err != nil {
				log.Println("Error stopping logger service:", err)
				return err
			}
			log.Println("Logger service stopped")
			return nil
		},
	})
}
