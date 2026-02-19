package preprocessing

import (
	"context"
	"fmt"

	"github.com/zeile/tui/internal/domain"
)

type Input struct {
	BookID      string
	SourcePath  string
	ManagedPath string
	CacheDir    string
}

type Result struct {
	Title    string
	Author   string
	Metadata string
}

type Processor interface {
	Process(ctx context.Context, input Input, onProgress func(stage string, percent float64)) (Result, error)
}

type Registry struct {
	processors map[domain.BookFormat]Processor
}

func NewRegistry() *Registry {
	return &Registry{processors: map[domain.BookFormat]Processor{}}
}

func (r *Registry) Register(format domain.BookFormat, processor Processor) {
	r.processors[format] = processor
}

func (r *Registry) ForFormat(format domain.BookFormat) (Processor, error) {
	processor, ok := r.processors[format]
	if !ok {
		return nil, fmt.Errorf("no processor registered for %s", format)
	}
	return processor, nil
}
