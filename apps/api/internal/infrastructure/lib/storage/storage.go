package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
)

type Object struct {
	Path string
	URL  string
	Size int64
}

type Storage interface {
	Save(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*Object, error)
	Delete(ctx context.Context, key string) error
}

func NewStorage(cfg config.FileStorageConfig) (Storage, error) {
	switch cfg.Provider {
	case "local":
		return NewLocalStorage(cfg.Local.BaseDir, cfg.Local.PublicPath), nil
	case "s3":
		return NewS3Storage(cfg.S3)
	default:
		return nil, fmt.Errorf("unsupported file storage provider: %s", cfg.Provider)
	}
}
