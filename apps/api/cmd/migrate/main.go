package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/database"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/logger"
)

const DefaultTimeout = 60 * time.Second

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	loggerService := logger.NewLoggerService(cfg.Observability)
	defer loggerService.Shutdown()

	log := logger.NewLoggerWithService(cfg.Observability, loggerService)

	timeout := cfg.Seeder.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	migrateCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := database.Migrate(migrateCtx, &log, cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to migrate database")
	}

	log.Info().Msg("database migrations completed")
}
