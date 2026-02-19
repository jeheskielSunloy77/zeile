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
	BookFormatPDF  BookFormat = "pdf"
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
	case ".pdf":
		return BookFormatPDF, nil
	default:
		return "", fmt.Errorf("unsupported format %q; only EPUB and PDF are supported", ext)
	}
}

func (f BookFormat) Extension() string {
	switch f {
	case BookFormatEPUB:
		return ".epub"
	case BookFormatPDF:
		return ".pdf"
	default:
		return ""
	}
}
