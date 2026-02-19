package preprocess

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeile/tui/internal/preprocessing"
)

type NoopProcessor struct{}

func (p NoopProcessor) Process(_ context.Context, input preprocessing.Input, onProgress func(stage string, percent float64)) (preprocessing.Result, error) {
	if onProgress != nil {
		onProgress("Preparing cache", 0.55)
	}

	if err := os.MkdirAll(input.CacheDir, 0o755); err != nil {
		return preprocessing.Result{}, fmt.Errorf("create cache dir: %w", err)
	}

	title := strings.TrimSuffix(filepath.Base(input.SourcePath), filepath.Ext(input.SourcePath))
	metadata := map[string]any{
		"source_path":  input.SourcePath,
		"managed_path": input.ManagedPath,
		"note":         "preprocessing pipeline initialized",
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return preprocessing.Result{}, fmt.Errorf("encode metadata: %w", err)
	}

	placeholder := filepath.Join(input.CacheDir, "manifest.json")
	if err := os.WriteFile(placeholder, metadataBytes, 0o644); err != nil {
		return preprocessing.Result{}, fmt.Errorf("write cache manifest: %w", err)
	}

	if onProgress != nil {
		onProgress("Cached", 0.9)
	}

	return preprocessing.Result{
		Title:    title,
		Author:   "Unknown",
		Metadata: string(metadataBytes),
	}, nil
}
