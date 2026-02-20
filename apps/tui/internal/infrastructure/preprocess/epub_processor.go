package preprocess

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/parser/epub"
	"github.com/zeile/tui/internal/preprocessing"
)

type EPUBProcessor struct{}

func (p EPUBProcessor) Process(ctx context.Context, input preprocessing.Input, onProgress func(stage string, percent float64)) (preprocessing.Result, error) {
	if err := ctx.Err(); err != nil {
		return preprocessing.Result{}, err
	}

	if onProgress != nil {
		onProgress("Parsing EPUB", 0.55)
	}

	cache, err := epub.Extract(ctx, input.ManagedPath)
	if err != nil {
		return preprocessing.Result{}, err
	}

	if err := ctx.Err(); err != nil {
		return preprocessing.Result{}, err
	}

	if onProgress != nil {
		onProgress("Writing EPUB cache", 0.8)
	}

	if err := os.MkdirAll(input.CacheDir, 0o755); err != nil {
		return preprocessing.Result{}, fmt.Errorf("create cache directory: %w", err)
	}

	cacheBytes, err := json.Marshal(cache)
	if err != nil {
		return preprocessing.Result{}, fmt.Errorf("encode epub cache: %w", err)
	}

	if err := os.WriteFile(epubCachePath(input.CacheDir), cacheBytes, 0o644); err != nil {
		return preprocessing.Result{}, fmt.Errorf("write epub cache file: %w", err)
	}

	meta, err := json.Marshal(map[string]any{
		"format":          domain.BookFormatEPUB,
		"sections":        len(cache.Sections),
		"cache_file":      epubCacheFile,
		"source_filename": filepath.Base(input.SourcePath),
	})
	if err != nil {
		return preprocessing.Result{}, fmt.Errorf("encode metadata: %w", err)
	}

	if onProgress != nil {
		onProgress("EPUB ready", 0.95)
	}

	result := preprocessing.Result{
		Title:    defaultTitle(cache.Title, input.SourcePath),
		Author:   defaultAuthor(cache.Author),
		Metadata: string(meta),
	}
	return result, nil
}

func defaultTitle(value, sourcePath string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
}

func defaultAuthor(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return "Unknown"
}
