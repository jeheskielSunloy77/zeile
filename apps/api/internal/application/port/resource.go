package port

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
)

type ResourceRepository[T domain.BaseModel] interface {
	Store(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id uuid.UUID, preloads []string) (*T, error)
	GetMany(ctx context.Context, opts GetManyOptions) ([]T, int64, error)
	Update(ctx context.Context, entity T, updates ...map[string]any) (*T, error)
	Destroy(ctx context.Context, id uuid.UUID) error
	Kill(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (*T, error)
	CacheEnabled() bool
	EvictCache(ctx context.Context, id uuid.UUID)
}

type JoinClause struct {
	Query string
	Args  []any
}

type WhereClause struct {
	Query string
	Args  []any
}

type GetManyOptions struct {
	Filters        map[string]any
	Joins          []JoinClause
	Wheres         []WhereClause
	Preloads       []string
	OrderBy        string
	OrderDirection string
	Limit          int
	Offset         int
}

func (o *GetManyOptions) Normalize() {
	if o.Limit <= 0 {
		o.Limit = 20
	}

	o.OrderDirection = strings.ToLower(strings.TrimSpace(o.OrderDirection))
	if o.OrderDirection == "" || (o.OrderDirection != "asc" && o.OrderDirection != "desc") {
		o.OrderDirection = "desc"
	}

	if o.OrderBy == "" {
		o.OrderBy = "created_at"
	}
}

func ParsePreloads(raw string) []string {
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	preloads := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		preloads = append(preloads, name)
	}

	if len(preloads) == 0 {
		return nil
	}
	return preloads
}
