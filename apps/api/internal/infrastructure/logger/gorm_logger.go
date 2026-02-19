package logger

import (
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/rs/zerolog"
	gormlogger "gorm.io/gorm/logger"
)

// gormWriter adapts zerolog to gorm's writer interface.
type gormWriter struct {
	logger *zerolog.Logger
}

func (w *gormWriter) Printf(format string, args ...any) {
	if w == nil || w.logger == nil {
		return
	}
	w.logger.Info().Msgf(format, args...)
}

// NewGormLogger creates a gorm-compatible logger backed by zerolog.
func NewGormLogger(base *zerolog.Logger, obsCfg *config.ObservabilityConfig) gormlogger.Interface {
	cfg := obsCfg
	if cfg == nil {
		cfg = config.DefaultObservabilityConfig()
	}

	logLevel := gormlogger.Error
	if base != nil {
		switch base.GetLevel() {
		case zerolog.DebugLevel:
			logLevel = gormlogger.Info
		case zerolog.InfoLevel:
			logLevel = gormlogger.Warn
		case zerolog.WarnLevel, zerolog.ErrorLevel:
			logLevel = gormlogger.Error
		default:
			logLevel = gormlogger.Error
		}
	}

	// Be more verbose locally for easier debugging.
	if cfg.Env == "local" && logLevel < gormlogger.Info {
		logLevel = gormlogger.Info
	}

	return gormlogger.New(&gormWriter{logger: base}, gormlogger.Config{
		SlowThreshold:             cfg.Logging.SlowQueryThreshold,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
		LogLevel:                  logLevel,
	})
}
