package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/jeheskielSunloy77/zeile/internal/application"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/database"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/cache"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/logger"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/repository"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/server"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/handler"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/router"
)

const DefaultContextTimeout = 30

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Initialize New Relic logger service
	loggerService := logger.NewLoggerService(cfg.Observability)
	defer loggerService.Shutdown()

	log := logger.NewLoggerWithService(cfg.Observability, loggerService)

	if cfg.Primary.Env != config.EnvDevelopment {
		log.Info().Msg(fmt.Sprintf("environment is not %s, running database migrations...", config.EnvDevelopment))
		if err := database.Migrate(context.Background(), &log, cfg); err != nil {
			log.Fatal().Err(err).Msg("failed to migrate database")
		}
	}

	// Initialize http server
	httpServer, err := server.NewServer(cfg, &log, loggerService)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize server")
	}

	cacheClient := cache.NewRedisCache(
		httpServer.Redis,
		&cfg.Cache,
		&log,
	)

	// Initialize repositories, services, and handlers
	repos := repository.NewRepositories(httpServer, cacheClient)
	services, serviceErr := application.NewServices(httpServer, repos)
	if serviceErr != nil {
		log.Fatal().Err(serviceErr).Msg("could not create services")
	}
	handlers := handler.NewHandlers(httpServer, services)

	// Initialize router
	r := router.NewRouter(httpServer, handlers, services)

	// Setup HTTP server
	httpServer.SetupFiber(r)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	// Start server
	go func() {
		if err = httpServer.Start(); err != nil {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), DefaultContextTimeout*time.Second)

	if err = httpServer.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}
	stop()
	cancel()

	log.Info().Msg("server exited properly")
}
