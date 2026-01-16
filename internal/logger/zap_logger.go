package logger

import (
	"geoNotifications/internal/config"
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogEvent struct {
	Level  zapcore.Level
	Msg    string
	Fields []zap.Field
}

func provideLogger(cfg *config.App) *zap.Logger {
	logLevel := parseLogLevel(cfg.Logger.Level)
	var Logger *zap.Logger
	switch cfg.Logger.Mode {
	case "prod", "production":
		writer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   "logs/app.log",
			MaxSize:    50,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		})

		encoderCfg := zap.NewProductionEncoderConfig()
		encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			writer,
			logLevel,
		)
		Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	default:
		cfg := zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(logLevel)
		var err error
		Logger, err = cfg.Build()
		if err != nil {
			log.Fatal(err)
		}
	}
	return Logger
}

func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn", "warning":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "dpanic":
		return zap.DPanicLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}
