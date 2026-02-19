package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/database"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/job"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/storage"
	loggerPkg "github.com/jeheskielSunloy77/zeile/internal/infrastructure/logger"
	"github.com/newrelic/go-agent/v3/integrations/nrredis-v9"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Server struct {
	Config        *config.Config
	Logger        *zerolog.Logger
	LoggerService *loggerPkg.LoggerService
	DB            *database.Database
	Redis         *redis.Client
	Job           *job.JobService
	Storage       storage.Storage
	App           *fiber.App
}

func NewServer(cfg *config.Config, logger *zerolog.Logger, loggerService *loggerPkg.LoggerService) (*Server, error) {
	db, err := database.NewDatabase(cfg, logger, loggerService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	storageProvider, err := storage.NewStorage(cfg.FileStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Redis client with New Relic integration
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.Cache.RedisAddress,
	})

	// Add New Relic Redis hooks if available
	if loggerService != nil && loggerService.GetApplication() != nil {
		redisClient.AddHook(nrredis.NewHook(redisClient.Options()))
	}

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("fail to connect to redis: %w", err)
	}

	// job service
	jobService := job.NewJobService(logger, cfg, db.DB, storageProvider)
	jobService.InitHandlers(cfg, logger)

	// Start job server
	if err := jobService.Start(); err != nil {
		return nil, err
	}

	server := &Server{
		Config:        cfg,
		Logger:        logger,
		LoggerService: loggerService,
		DB:            db,
		Redis:         redisClient,
		Job:           jobService,
		Storage:       storageProvider,
	}

	// Start metrics collection
	// Runtime metrics are automatically collected by New Relic Go agent

	return server, nil
}

func (s *Server) SetupFiber(app *fiber.App) {
	s.App = app
}

func (s *Server) Start() error {
	if s.App == nil {
		return errors.New("fiber app not initialized")
	}

	s.Logger.Info().
		Str("port", s.Config.Server.Port).
		Str("env", string(s.Config.Primary.Env)).
		Msg("starting server")

	return s.App.Listen(":" + s.Config.Server.Port)
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownErr := func() error {
		if s.App == nil {
			return nil
		}
		done := make(chan error, 1)
		go func() {
			done <- s.App.Shutdown()
		}()

		select {
		case err := <-done:
			if err != nil {
				return fmt.Errorf("failed to shutdown HTTP server: %w", err)
			}
		case <-ctx.Done():
			return fmt.Errorf("failed to shutdown HTTP server: %w", ctx.Err())
		}

		return nil
	}()

	if shutdownErr != nil {
		return shutdownErr
	}

	if err := s.DB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	if s.Job != nil {
		s.Job.Stop()
	}

	return nil
}
