# GeoNotifications Core (Go)

Ядро системы геооповещений: принимает координаты мобильного приложения, возвращает активные инциденты и при опасности асинхронно шлет вебхук на портал новостей.

## Архитектура
- Clean Architecture: Handlers (Gin) → Services → Storage (Postgres/Redis) → внешние очереди/вебхуки.
- Асинхронный логгер (zap) и очередь повторов для кэша/вебхуков.
- Swagger доступен по `/api/v1/swagger/index.html`.


## Состав репозитория

- `cmd/geoNotifications/main.go` — точка входа сервиса (Fx DI).
- `cmd/webhook-stub/` — заглушка вебхука (порт 9090, Dockerfile + entrypoint).
- `internal/`
  - `app/` — запуск приложения и DI провайдеры.
  - `config/` — модели конфигурации и загрузка из env.
  - `domain/` — сущности и доменные ошибки.
  - `logger/` — zap-логгер и асинхронная обертка.
  - `rerty/` — стратегия retry.
  - `service/` — бизнес-логика (incidents, location, system).
  - `storage/postgres/` — Postgres-репозитории.
  - `storage/redis/` — кэш и очередь для ретраев.
  - `web/` — dto, хендлеры Gin, middleware, роутеры.
  - `webhookSender/` — отправка вебхуков.
- `docs/` — Swagger-описание.
- `migrations/` — SQL-миграции.
- `docker-compose.yaml` — Postgres, Redis, webhook stub.
- `go.mod`, `go.sum` — зависимости.


## Требования
- Go 1.24+
- Postgres 15
- Redis 6+
- Docker / docker-compose (для локального поднятия стека)

## Быстрый старт (Docker)

Для работы ngrok (если нужно пробросить вебхук-заглушку):
В .env задайте `WEBHOOK_URL` вида `https://<your_subdomain>.ngrok.io/webhook`.
Перед этим задайте `NGROK_AUTHTOKEN` (зарегистрируйтесь на ngrok.com).
```bash
cp .env.example .env   # если есть, иначе задайте переменные вручную
docker-compose up --build
go run ./cmd/geoNotifications
```
По умолчанию сервис слушает `${SERVER_PORT}`. Вебхук-заглушка поднимется на :9090 (cmd/webhook-stub).
Контейнеры: postgres → порт 5433, Redis → 6379
Миграции применятся автоматически при старте сервиса.
Ngrok + webhook сервис также запустся автоматически, если задан `NGROK_AUTHTOKEN`.

## Миграции
SQL-миграции лежат в `migrations/`. 
```bash
migrate -path ./migrations -database "postgres://user:password@localhost:5433/dbname?
```
## Тесты и покрытие
```bash
go test ./... -cover
```

## Основные эндпоинты
Private (API-key в `X-API-Key`):
- `POST /api/v1/incidents` — создать .
- `GET /api/v1/incidents?page=1&limit=10` — список с пагинацией.
- `GET /api/v1/incidents/{id}` — получить по ID.
- `PUT /api/v1/incidents/{id}` — обновить.
- `DELETE /api/v1/incidents/{id}` — деактивировать.
Public:
- `GET /api/v1/incidents/stats` — статистика (уникальные user_id за окно времени).
- `POST /api/v1/location/check` — публичная проверка координат; сохраняет историю и ставит задачу на вебхук.
- `GET /api/v1/system/health` — health-check.
- Swagger UI: `GET /api/v1/swagger/index.html`.

## Вебхуки
- Асинхронная отправка через `internal/webhookSender`.
- Тестовая заглушка: `cmd/webhook-stub` (порт 9090). Можно туннелировать через ngrok: `ngrok http 9090` и передать URL в `WEBHOOK_URL`.

## Логирование и наблюдаемость
- zap, асинхронный буфер `LOGGER_BUFFER_SIZE`.
- Middleware логирует HTTP-запросы.
