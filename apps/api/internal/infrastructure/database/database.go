package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	loggerConfig "github.com/jeheskielSunloy77/zeile/internal/infrastructure/logger"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	DB    *gorm.DB
	SQLDB *sql.DB
	log   *zerolog.Logger
}

const DatabasePingTimeout = 10

func NewDatabase(cfg *config.Config, logger *zerolog.Logger, loggerService *loggerConfig.LoggerService) (*Database, error) {
	dsn := buildDSN(cfg)

	obsCfg := cfg.Observability
	if obsCfg == nil {
		obsCfg = config.DefaultObservabilityConfig()
		obsCfg.Env = cfg.Primary.Env
	}

	gormLogger := loggerConfig.NewGormLogger(logger, obsCfg)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DriverName: "postgres",
		DSN:        dsn,
	}), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gorm database: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database handle: %w", err)
	}
	applyPoolSettings(sqlDB, cfg)

	database := &Database{
		DB:    gormDB,
		SQLDB: sqlDB,
		log:   logger,
	}

	ctx, cancel := context.WithTimeout(context.Background(), DatabasePingTimeout*time.Second)
	defer cancel()
	if err = sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().Msg("connected to the database")

	return database, nil
}

func (db *Database) Close() error {
	if db.log != nil {
		db.log.Info().Msg("closing database connection pool")
	}

	if db.SQLDB != nil {
		return db.SQLDB.Close()
	}

	return nil
}

func applyPoolSettings(db *sql.DB, cfg *config.Config) {
	if db == nil || cfg == nil {
		return
	}

	if cfg.Database.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	}
	if cfg.Database.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)
	}
}

func buildDSN(cfg *config.Config) string {
	hostPort := net.JoinHostPort(cfg.Database.Host, strconv.Itoa(cfg.Database.Port))

	encodedPassword := url.QueryEscape(cfg.Database.Password)
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.Database.User,
		encodedPassword,
		hostPort,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
}
