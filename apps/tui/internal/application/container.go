package application

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/config"
	"github.com/zeile/tui/internal/infrastructure/database"
	"github.com/zeile/tui/internal/infrastructure/preprocess"
	"github.com/zeile/tui/internal/infrastructure/repository"
	"github.com/zeile/tui/internal/infrastructure/storage"
	"github.com/zeile/tui/internal/preprocessing"
)

type Container struct {
	Config  config.Config
	Paths   storage.Paths
	DB      *sql.DB
	Library *LibraryService
}

func NewContainer(ctx context.Context) (*Container, error) {
	loadedConfig, err := config.Load()
	if err != nil {
		return nil, err
	}

	paths := storage.Resolve(loadedConfig.Config, loadedConfig.Path)
	if err := paths.Ensure(); err != nil {
		return nil, err
	}

	db, err := database.Open(paths.DBPath)
	if err != nil {
		return nil, err
	}

	if err := database.Migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	bookRepo := repository.NewBookRepository(db)
	stateRepo := repository.NewReadingStateRepository(db)

	processorRegistry := preprocessing.NewRegistry()
	processorRegistry.Register(domain.BookFormatEPUB, preprocess.NoopProcessor{})
	processorRegistry.Register(domain.BookFormatPDF, preprocess.NoopProcessor{})

	library := NewLibraryService(bookRepo, stateRepo, processorRegistry, paths)

	return &Container{
		Config:  loadedConfig.Config,
		Paths:   paths,
		DB:      db,
		Library: library,
	}, nil
}

func (c *Container) Close() error {
	if c == nil || c.DB == nil {
		return nil
	}
	if err := c.DB.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}
	return nil
}
