# GeoNotifications Core (Go)

GeoNotifications - бэкенд-сервис системы геооповещений. Он решает одну конкретную задачу: в реальном времени определять, находится ли пользователь 
мобильного приложения в зоне опасного инцидента (пожар, авария, стихийное бедствие), и если да - немедленно уведомлять об этом портал новостей через webhook.

### Архитектура

- Clean Architecture: HTTP-хендлеры (Gin) -> сервисы (бизнес-логика) -> хранилища (Postgres, Redis) -> внешние системы (webhook).
- Ручная сборка зависимостей в `internal/di/di.go` -- без DI-контейнеров и магии. Все ошибки конфигурации отлавливаются при старте, а не в рантайме.
- Graceful shutdown: при получении SIGINT/SIGTERM сервис последовательно останавливает HTTP-сервер, воркеры очередей, закрывает соединения с Redis и Postgres, завершает логгер.
- Асинхронный логгер на основе zap с буферизацией.
- PostgreSQL с расширением PostGIS. Используется функция `ST_DWithin`
- Инциденты кэшируются в Redis. При чтении сервис сначала ищет данные в кэше, при промахе -- идет в PostgreSQL и прогревает кэш.
- Retry стратегия
- Webhook-очередь

## Состав репозитория

```
cmd/
  geoNotifications/main.go   - точка входа в сервис
  webhook-stub/               - webhook-приемник (Dockerfile + entrypoint)

internal/
  app/                        - запуск приложения, обработка сигналов, graceful shutdown
  config/                     - конфиг, переменные окружения, валидация
  di/                         -- явная сборка (Container)
  domain/                     - доменные сущности (Incident, Location, LocationCheckTask) и ошибки
  logger/                     - zap-логгер с асинхронной оберткой
  retry/                      - retry стратегия (attempts, delay, backoff)
  service/                    - бизнес-логика: incidents, location, system
  storage/postgres/           - репозитории PostgreSQL (PostGIS для пространственных запросов)
  storage/redis/              - кэш инцидентов, очередь webhook-задач, очередь retry для кэша
  web/dto/                    - структуры запросов/ответов
  web/handlers/               - HTTP-хендлеры Gin
  web/middlewares/            - middleware (логирование запросов, аутентификация по API-ключу)
  web/routers/                - роутинг (публичные и приватные эндпоинты)
  webhook/                    - отправка HTTP POST на внешний webhook

docs/                         - Swagger
migrations/                   - SQL-миграции (создание таблиц, PostGIS-расширение, GIST-индекс)
docker-compose.yaml           - PostGIS, Redis, миграции, webhook-заглушка, ngrok
```

## Требования

- Go 1.24+
- PostgreSQL 16 с PostGIS
- Redis 7+
- Docker и docker compose (для локального поднятия инфраструктуры)

## Быстрый старт

```bash
# 1. Скопировать и заполнить переменные окружения
cp .env.example .env

# 2. Поднять инфраструктуру (PostgreSQL + PostGIS, Redis, миграции, webhook-заглушка)
docker compose up --build -d

# 3. Запустить сервис
go run ./cmd/geoNotifications
```

Сервис запустится на `${SERVER_HOST}:${SERVER_PORT}` (по умолчанию `:8080`).

Контейнеры:
- PostgreSQL (PostGIS) -- порт 5433
- Redis -- порт 6379
- Webhook-заглушка -- порт 9090
- Ngrok (опционально) -- веб-интерфейс на порту 4040

Миграции применяются автоматически через контейнер `migrate`.

Если нужен внешний доступ к webhook-заглушке, задайте `NGROK_AUTHTOKEN` в `.env` и укажите `WEBHOOK_URL` вида `https://<subdomain>.ngrok.io/webhook`.

## Миграции

SQL-миграции лежат в `migrations/`. Для ручного запуска:

```bash
migrate -path ./migrations \
  -database "postgres://user:password@localhost:5433/dbname?sslmode=disable" up
```

## API

### Публичные эндпоинты

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/location/check` | Проверка координат пользователя. Возвращает список инцидентов, в зоне которых он находится. Сохраняет запись в историю и при опасности ставит задачу на webhook. |
| GET | `/api/v1/incidents/stats` | Статистика: количество уникальных пользователей, проверивших координаты за настроенное временное окно. |
| GET | `/api/v1/system/health` | Health-check (проверка доступности PostgreSQL и Redis). |
| GET | `/api/v1/swagger/index.html` | Swagger UI. |

### Приватные эндпоинты (требуется заголовок `X-API-Key`)

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/incidents` | Создать инцидент (координаты, радиус, severity, тип). |
| GET | `/api/v1/incidents?page=1&limit=10` | Список инцидентов с пагинацией. |
| GET | `/api/v1/incidents/{id}` | Получить инцидент по ID. |
| PUT | `/api/v1/incidents/{id}` | Обновить инцидент. |
| DELETE | `/api/v1/incidents/{id}` | Деактивировать инцидент (soft delete). |

## Тесты

```bash
go test ./... -cover
```

## Переменные окружения

Основные переменные (полный список -- в `internal/config/config_model.go`):

| Переменная | Описание |
|------------|----------|
| `SERVER_HOST`, `SERVER_PORT` | Адрес и порт HTTP-сервера |
| `DB_POSTGRES_HOST`, `DB_POSTGRES_PORT`, `DB_POSTGRES_USER`, `DB_POSTGRES_PASSWORD`, `DB_POSTGRES_DBNAME`, `DB_POSTGRES_SSLMODE` | Подключение к PostgreSQL |
| `DB_REDIS_HOST`, `DB_REDIS_PORT`, `DB_REDIS_PASSWORD`, `DB_REDIS_DB` | Подключение к Redis |
| `DB_REDIS_CACHESIZE` | Максимальный размер кэша инцидентов |
| `DB_REDIS_CACHERETRYSIZE` | Размер буфера очереди повторных попыток записи в кэш |
| `DB_TIMEOUTS_WRITE`, `DB_TIMEOUTS_READ`, `DB_TIMEOUTS_LONG` | Таймауты операций с БД |
| `WEBHOOK_URL` | URL внешнего webhook-приемника |
| `AUTH_APIKEY` | API-ключ для приватных эндпоинтов |
| `RETRY_ATTEMPTS`, `RETRY_DELAY`, `RETRY_BACKOFF` | Параметры стратегии повторных попыток |
| `STATS_TIME_WINDOW_MINUTES` | Временное окно для подсчета статистики (в минутах) |
| `INCIDENT_TITLEMINLENGTH`, `INCIDENT_TITLEMAXLENGTH` | Ограничения длины заголовка инцидента |
| `INCIDENT_DESCRMINLENGTH`, `INCIDENT_DESCRMAXLENGTH` | Ограничения длины описания инцидента |
| `LOGGER_MODE`, `LOGGER_LEVEL`, `LOGGER_BUFFER_SIZE` | Настройки логгера |
| `GIN_MODE` | Режим Gin (`debug`, `release`, `test`) |
| `NGROK_AUTHTOKEN` | Токен ngrok для туннелирования webhook-заглушки (опционально) |
