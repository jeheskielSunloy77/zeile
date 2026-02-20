package files

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFingerprintSHA256ContextCanceled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.bin")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := FingerprintSHA256Context(ctx, path); err == nil {
		t.Fatalf("expected canceled error")
	}
}

func TestCopyFileContextCanceled(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	if err := os.WriteFile(src, []byte("copy me"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := CopyFileContext(ctx, src, dst); err == nil {
		t.Fatalf("expected canceled error")
	}
}
