package application

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/config"
	"github.com/zeile/tui/internal/infrastructure/database"
	"github.com/zeile/tui/internal/infrastructure/preprocess"
	"github.com/zeile/tui/internal/infrastructure/remote"
	"github.com/zeile/tui/internal/infrastructure/repository"
	"github.com/zeile/tui/internal/infrastructure/storage"
	"github.com/zeile/tui/internal/preprocessing"
)

type Container struct {
	Config  config.Config
	Paths   storage.Paths
	DB      *sql.DB
	Auth    *AuthService
	Sync    *SyncService
	Library *LibraryService
	Reader  *ReaderService
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
	processorRegistry.Register(domain.BookFormatEPUB, preprocess.EPUBProcessor{})
	processorRegistry.Register(domain.BookFormatPDF, preprocess.PDFProcessor{})

	library := NewLibraryService(bookRepo, stateRepo, processorRegistry, paths)
	readerService := NewReaderService(bookRepo, stateRepo, paths)
	authService, err := NewAuthService(loadedConfig.Config, paths)
	if err != nil {
		db.Close()
		return nil, err
	}
	syncRepo := repository.NewSyncRepository(db)
	syncService := NewSyncService(
		authService,
		library,
		syncRepo,
		syncRepo,
		remote.NewClient(loadedConfig.Config.APIBaseURL),
	)

	return &Container{
		Config:  loadedConfig.Config,
		Paths:   paths,
		DB:      db,
		Auth:    authService,
		Sync:    syncService,
		Library: library,
		Reader:  readerService,
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
