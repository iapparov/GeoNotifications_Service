package config

import "time"

type App struct {
	Server    Server
	Logger    Logger
	DB        DB
	Gin       Gin
	Auth      Auth
	Incident  Incident
	WebHook   string `env:"WEBHOOK_URL"`
	StatsTime int    `env:"STATS_TIME_WINDOW_MINUTES"`
	Retry     Retry
}

type Incident struct {
	TitleMinLength       int `env:"INCIDENT_TITLEMINLENGTH"`
	TitleMaxLength       int `env:"INCIDENT_TITLEMAXLENGTH"`
	DescriptionMinLength int `env:"INCIDENT_DESCRMINLENGTH"`
	DescriptionMaxLength int `env:"INCIDENT_DESCRMAXLENGTH"`
}

type Retry struct {
	MaxAttempts int           `env:"RETRY_ATTEMPTS"`
	Delay       time.Duration `env:"RETRY_DELAY"`
	Backoff     float64       `env:"RETRY_BACKOFF"`
}

type Auth struct {
	ApiKey string `env:"AUTH_APIKEY"`
}

type Gin struct {
	Mode string `env:"GIN_MODE"`
}

type Server struct {
	Host string `env:"SERVER_HOST"`
	Port int    `env:"SERVER_PORT"`
}

type Logger struct {
	Mode     string `env:"LOGGER_MODE"`
	Level    string `env:"LOGGER_LEVEL"`
	BuffSize int    `env:"LOGGER_BUFFER_SIZE"`
}

type DB struct {
	Postgres Postgres
	Redis    Redis
	TimeOuts TimeOuts
}

type TimeOuts struct {
	Write time.Duration `env:"DB_TIMEOUTS_WRITE"`
	Read  time.Duration `env:"DB_TIMEOUTS_READ"`
	Long  time.Duration `env:"DB_TIMEOUTS_LONG"`
}

type Postgres struct {
	Host            string        `env:"DB_POSTGRES_HOST"`
	Port            int           `env:"DB_POSTGRES_PORT"`
	User            string        `env:"DB_POSTGRES_USER"`
	Password        string        `env:"DB_POSTGRES_PASSWORD"`
	DBName          string        `env:"DB_POSTGRES_DBNAME"`
	SSLMode         string        `env:"DB_POSTGRES_SSLMODE"`
	MaxOpenConns    int           `env:"DB_POSTGRES_MAXOPENCONNS"`
	MaxIdleConns    int           `env:"DB_POSTGRES_MAXIDLECONNS"`
	ConnMaxLifetime time.Duration `env:"DB_POSTGRES_CONNMAXLIFETIME"`
}

type Redis struct {
	Host           string `env:"DB_REDIS_HOST"`
	Port           int    `env:"DB_REDIS_PORT"`
	Password       string `env:"DB_REDIS_PASSWORD"`
	DB             int    `env:"DB_REDIS_DB"`
	CacheSize      int    `env:"DB_REDIS_CACHESIZE"`
	CacheRetrySize int    `env:"DB_REDIS_CACHERETRYSIZE"`
}
