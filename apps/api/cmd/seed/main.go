package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/database"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/database/seeder"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	if cfg.Seeder.Enabled == false {
		panic("seeder is disabled in configuration")
	}

	users := flag.Int("users", seeder.DefaultUserCount, "number of users to seed")
	migrate := flag.Bool("migrate", false, "run migrations before seeding")
	flag.Parse()

	loggerService := logger.NewLoggerService(cfg.Observability)
	defer loggerService.Shutdown()

	log := logger.NewLoggerWithService(cfg.Observability, loggerService)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if *migrate {
		migrateCtx, cancel := context.WithTimeout(ctx, cfg.Seeder.Timeout)
		if err := database.Migrate(migrateCtx, &log, cfg); err != nil {
			log.Fatal().Err(err).Msg("failed to migrate database")
		}
		cancel()
	}

	db, err := database.NewDatabase(cfg, &log, loggerService)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("failed to close database")
		}
	}()

	seedCtx, cancel := context.WithTimeout(ctx, cfg.Seeder.Timeout)
	defer cancel()

	opts := seeder.Options{UsersCount: *users}
	if err := seeder.Run(seedCtx, db.DB, &log, opts); err != nil {
		log.Fatal().Err(err).Msg("seeding failed")
	}

	log.Info().Str("status", "ok").Msg("seeding finished")
}
