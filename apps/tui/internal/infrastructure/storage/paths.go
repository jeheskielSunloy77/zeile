package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/config"
)

type Paths struct {
	BaseDir    string
	LibraryDir string
	CacheDir   string
	DBPath     string
	ConfigPath string
}

func Resolve(cfg config.Config, configPath string) Paths {
	base := cfg.DataDir
	return Paths{
		BaseDir:    base,
		LibraryDir: filepath.Join(base, "library"),
		CacheDir:   filepath.Join(base, "cache"),
		DBPath:     filepath.Join(base, "zeile.db"),
		ConfigPath: configPath,
	}
}

func (p Paths) Ensure() error {
	dirs := []string{p.BaseDir, p.LibraryDir, p.CacheDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}

func (p Paths) ManagedBookPath(bookID string, format domain.BookFormat) string {
	return filepath.Join(p.LibraryDir, bookID+format.Extension())
}

func (p Paths) BookCacheDir(bookID string) string {
	return filepath.Join(p.CacheDir, bookID)
}
