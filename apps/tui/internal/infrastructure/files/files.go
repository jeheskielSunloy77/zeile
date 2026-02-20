package files

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func FingerprintSHA256(path string) (string, error) {
	return FingerprintSHA256Context(context.Background(), path)
}

func FingerprintSHA256Context(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for fingerprint: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	buffer := make([]byte, 128*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		read, readErr := file.Read(buffer)
		if read > 0 {
			if _, err := hasher.Write(buffer[:read]); err != nil {
				return "", fmt.Errorf("hash file: %w", err)
			}
		}
		if readErr == nil {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		return "", fmt.Errorf("hash file: %w", readErr)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func CopyFile(srcPath, dstPath string) error {
	return CopyFileContext(context.Background(), srcPath, dstPath)
}

func CopyFileContext(ctx context.Context, srcPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer dst.Close()

	buffer := make([]byte, 128*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		read, readErr := src.Read(buffer)
		if read > 0 {
			if _, err := dst.Write(buffer[:read]); err != nil {
				return fmt.Errorf("copy file: %w", err)
			}
		}
		if readErr == nil {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		return fmt.Errorf("copy file: %w", readErr)
	}

	if err := dst.Sync(); err != nil {
		return fmt.Errorf("sync destination file: %w", err)
	}

	return nil
}
