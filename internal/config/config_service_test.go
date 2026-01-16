package config

import (
	"os"
	"testing"
)

func setTestEnv() {
	envs := map[string]string{
		"SERVER_HOST":                 "localhost",
		"SERVER_PORT":                 "8080",
		"LOGGER_MODE":                 "dev",
		"LOGGER_LEVEL":                "debug",
		"LOGGER_BUFFER_SIZE":          "10",
		"GIN_MODE":                    "debug",
		"AUTH_APIKEY":                 "secret",
		"INCIDENT_TITLEMINLENGTH":     "1",
		"INCIDENT_TITLEMAXLENGTH":     "100",
		"INCIDENT_DESCRMINLENGTH":     "1",
		"INCIDENT_DESCRMAXLENGTH":     "200",
		"RETRY_ATTEMPTS":              "2",
		"RETRY_DELAY":                 "1s",
		"RETRY_BACKOFF":               "2",
		"DB_TIMEOUTS_WRITE":           "1s",
		"DB_TIMEOUTS_READ":            "1s",
		"DB_TIMEOUTS_LONG":            "2s",
		"DB_POSTGRES_HOST":            "pg",
		"DB_POSTGRES_PORT":            "5432",
		"DB_POSTGRES_USER":            "user",
		"DB_POSTGRES_PASSWORD":        "pass",
		"DB_POSTGRES_DBNAME":          "db",
		"DB_POSTGRES_SSLMODE":         "disable",
		"DB_POSTGRES_MAXOPENCONNS":    "10",
		"DB_POSTGRES_MAXIDLECONNS":    "5",
		"DB_POSTGRES_CONNMAXLIFETIME": "1m",
		"DB_REDIS_HOST":               "redis",
		"DB_REDIS_PORT":               "6379",
		"DB_REDIS_PASSWORD":           "",
		"DB_REDIS_DB":                 "0",
		"DB_REDIS_CACHESIZE":          "10",
		"DB_REDIS_CACHERETRYSIZE":     "10",
		"WEBHOOK_URL":                 "http://example.com",
		"STATS_TIME_WINDOW_MINUTES":   "5",
	}
	for k, v := range envs {
		_ = os.Setenv(k, v)
	}
}

func TestNewAppConfig_ParseEnv(t *testing.T) {
	setTestEnv()
	cfg, err := NewAppConfig()
	if err != nil {
		t.Fatalf("expected config parse success, got %v", err)
	}
	if cfg.DB.Postgres.Host != "pg" || cfg.DB.Postgres.Port != 5432 {
		t.Fatalf("unexpected postgres config: %+v", cfg.DB.Postgres)
	}
	if cfg.Logger.Level != "debug" || cfg.Logger.BuffSize != 10 {
		t.Fatalf("unexpected logger config: %+v", cfg.Logger)
	}
	if cfg.Auth.ApiKey != "secret" {
		t.Fatalf("unexpected api key: %s", cfg.Auth.ApiKey)
	}
}
