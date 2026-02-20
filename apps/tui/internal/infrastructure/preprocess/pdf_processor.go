package preprocess

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeile/tui/internal/domain"
	pdfparser "github.com/zeile/tui/internal/infrastructure/parser/pdf"
	"github.com/zeile/tui/internal/preprocessing"
)

type PDFProcessor struct{}

func (p PDFProcessor) Process(ctx context.Context, input preprocessing.Input, onProgress func(stage string, percent float64)) (preprocessing.Result, error) {
	if err := ctx.Err(); err != nil {
		return preprocessing.Result{}, err
	}

	if onProgress != nil {
		onProgress("Parsing PDF", 0.55)
	}

	cache, err := pdfparser.Extract(ctx, input.ManagedPath)
	if err != nil {
		return preprocessing.Result{}, err
	}

	if err := ctx.Err(); err != nil {
		return preprocessing.Result{}, err
	}

	if onProgress != nil {
		onProgress("Writing PDF cache", 0.8)
	}

	if err := os.MkdirAll(input.CacheDir, 0o755); err != nil {
		return preprocessing.Result{}, fmt.Errorf("create cache directory: %w", err)
	}

	cacheBytes, err := json.Marshal(cache)
	if err != nil {
		return preprocessing.Result{}, fmt.Errorf("encode pdf cache: %w", err)
	}
	if err := os.WriteFile(pdfCachePath(input.CacheDir), cacheBytes, 0o644); err != nil {
		return preprocessing.Result{}, fmt.Errorf("write pdf cache file: %w", err)
	}

	meta, err := json.Marshal(map[string]any{
		"format":          domain.BookFormatPDF,
		"pages":           len(cache.Pages),
		"layout_pages":    len(cache.LayoutPages),
		"cache_file":      pdfCacheFile,
		"source_filename": filepath.Base(input.SourcePath),
	})
	if err != nil {
		return preprocessing.Result{}, fmt.Errorf("encode metadata: %w", err)
	}

	if onProgress != nil {
		onProgress("PDF ready", 0.95)
	}

	result := preprocessing.Result{
		Title:    defaultPDFTitle(cache.Title, input.SourcePath),
		Author:   defaultPDFAuthor(cache.Author),
		Metadata: string(meta),
	}
	return result, nil
}

func defaultPDFTitle(value, sourcePath string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
}

func defaultPDFAuthor(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return "Unknown"
}
