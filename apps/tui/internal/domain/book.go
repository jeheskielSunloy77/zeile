package domain

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type BookFormat string

const (
	BookFormatEPUB BookFormat = "epub"
)

type Book struct {
	ID          string
	Fingerprint string
	Title       string
	Author      string
	Format      BookFormat
	AddedAt     time.Time
	LastOpened  *time.Time
	SourcePath  string
	ManagedPath string
	Metadata    string
	SizeBytes   int64
}

func DetectFormat(path string) (BookFormat, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".epub":
		return BookFormatEPUB, nil
	default:
		return "", fmt.Errorf("unsupported format %q; only EPUB is supported", ext)
	}
}

func (f BookFormat) Extension() string {
	switch f {
	case BookFormatEPUB:
		return ".epub"
	default:
		return ""
	}
}
