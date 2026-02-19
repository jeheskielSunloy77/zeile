package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	_ "github.com/lib/pq"

	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrate(ctx context.Context, logger *zerolog.Logger, cfg *config.Config) error {
	db, err := sql.Open("postgres", buildDSN(cfg))
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("constructing database migrator: %w", err)
	}

	sourceDriver, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("loading database migrations: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("creating migrate instance: %w", err)
	}

	noChange := false
	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("running migrations: %w", err)
		}
		noChange = true
	}

	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			logger.Info().Msg("database schema initialized")
			return nil
		}
		return fmt.Errorf("retrieving migration version: %w", err)
	}

	status := "migrated database schema"
	if noChange {
		status = "database schema up to date"
	}

	logEvent := logger.Info()
	if dirty {
		logEvent = logger.Warn()
		status += " (dirty state detected)"
	}

	logEvent.Uint("version", version).Msg(status)
	return nil
}
