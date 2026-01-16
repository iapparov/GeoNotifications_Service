package app

import (
	"geoNotifications/internal/config"
	"geoNotifications/internal/di"
	"geoNotifications/internal/logger"
	"geoNotifications/internal/service"
	"geoNotifications/internal/storage/postgres"
	"geoNotifications/internal/storage/redis"
	"geoNotifications/internal/web/handlers"
	"geoNotifications/internal/webhookSender"

	"go.uber.org/fx"
)

func Run() {
	app := fx.New(
		fx.Provide(
			config.NewAppConfig,
			logger.NewService,
			postgres.NewPostgres,
			redis.NewRedis,
			func(db *postgres.Postgres) redis.DB {
				return db
			},
			redis.NewCacheRetryQueue,
			webhookSender.NewWebhookSender,
			service.NewIncidents,
			service.NewLocation,
			service.NewSystem,
			func(pg *postgres.Postgres) service.DBChecker { return pg },
			func(rdb *redis.Redis) service.RedisChecker { return rdb },
			func(pg *postgres.Postgres) service.IncidentsDB {
				return pg
			},
			func(rdb *redis.Redis) service.IncidentsCache { return rdb },
			func(rcq *redis.CacheRetryQueue) service.CacheRetryQueue { return rcq },
			func(pg *postgres.Postgres) service.LocationDB { return pg },
			func(rdb *redis.Redis) service.LocationCache { return rdb },
			func(rdb *redis.Redis) service.LocationQueue { return rdb },
			handlers.NewIncidents,
			handlers.NewLocation,
			handlers.NewSystem,
			func(inc *service.Incidents) handlers.IncidentsService { return inc },
			func(loc *service.Location) handlers.LocationService { return loc },
			func(sys *service.System) handlers.SystemService { return sys },
		),

		fx.Invoke(
			di.StartLoggerAsync,
			di.StartHttpServer,
			di.WarmUpRedisOnStart,
			di.StartWebhookService,
			di.CloseRedisOnStop,
			di.ClosePostgresOnStop,
			di.StartCacheRetryService,
		),
	)
	app.Run()
}
