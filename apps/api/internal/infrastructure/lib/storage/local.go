package storage

import (
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type LocalStorage struct {
	baseDir    string
	publicPath string
}

func NewLocalStorage(baseDir, publicPath string) *LocalStorage {
	base := strings.TrimSpace(baseDir)
	if base == "" {
		base = "static"
	}

	public := strings.TrimSpace(publicPath)
	if public == "" {
		public = "/static"
	}

	public = strings.TrimRight(public, "/")

	return &LocalStorage{
		baseDir:    base,
		publicPath: public,
	}
}

func (s *LocalStorage) Save(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*Object, error) {
	_ = ctx

	cleanKey := strings.TrimLeft(path.Clean("/"+key), "/")
	fullPath := filepath.Join(s.baseDir, filepath.FromSlash(cleanKey))

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return nil, err
	}

	out, err := os.Create(fullPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	if _, err := io.Copy(out, reader); err != nil {
		return nil, err
	}

	url := s.publicPath + "/" + cleanKey
	return &Object{Path: cleanKey, URL: url, Size: size}, nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	_ = ctx

	cleanKey := strings.TrimLeft(path.Clean("/"+key), "/")
	fullPath := filepath.Join(s.baseDir, filepath.FromSlash(cleanKey))
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
