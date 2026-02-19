package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/application/port"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/cache"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/utils"
	"gorm.io/gorm"
)

type ResourceRepository[T domain.BaseModel] = port.ResourceRepository[T]

type JoinClause = port.JoinClause
type WhereClause = port.WhereClause
type GetManyOptions = port.GetManyOptions

type resourceRepository[T domain.BaseModel] struct {
	cfg            *config.Config
	db             *gorm.DB
	cache          cache.Cache
	isCacheEnabled bool
}

func NewResourceRepository[T domain.BaseModel](cfg *config.Config, db *gorm.DB, cacheClient cache.Cache) ResourceRepository[T] {
	isCacheEnabled := cacheClient != nil && cfg.Cache.TTL > 0

	return &resourceRepository[T]{db: db, cache: cacheClient, cfg: cfg, isCacheEnabled: isCacheEnabled}
}

func (r *resourceRepository[T]) CacheEnabled() bool {
	return r.isCacheEnabled
}

func (r *resourceRepository[T]) EvictCache(ctx context.Context, id uuid.UUID) {
	if r.isCacheEnabled {
		_ = r.cache.Delete(ctx, utils.GetModelCacheKey[T](id))
	}
}

func (r *resourceRepository[T]) Store(ctx context.Context, entity *T) error {
	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return err
	}

	if r.isCacheEnabled {
		id := (*entity).GetID()
		_ = r.cache.SetJSON(ctx, utils.GetModelCacheKey[T](id), entity)
	}

	return nil
}

func (r *resourceRepository[T]) GetByID(ctx context.Context, id uuid.UUID, preloads []string) (*T, error) {
	if len(preloads) == 0 && r.isCacheEnabled {
		if cached, ok := r.getCachedByID(ctx, id); ok {
			return cached, nil
		}
	}

	var entity T
	query := applyPreloads(r.db.WithContext(ctx), preloads)
	if err := query.First(&entity, id).Error; err != nil {
		return nil, err
	}

	if r.isCacheEnabled {
		_ = r.cache.SetJSON(ctx, utils.GetModelCacheKey[T](id), &entity)
	}

	return &entity, nil
}

func (r *resourceRepository[T]) Update(ctx context.Context, entity T, updates ...map[string]any) (*T, error) {
	// if the updates are provided, use them to only update specific fields, if not replace the entire entity
	var err error
	if len(updates) > 0 {
		err = r.db.WithContext(ctx).Model(&entity).Updates(updates[0]).Error
	} else {
		err = r.db.WithContext(ctx).Save(&entity).Error
	}
	if err != nil {
		return nil, err
	}

	r.EvictCache(ctx, entity.GetID())

	// return updated entity
	return &entity, nil
}

func (r *resourceRepository[T]) Destroy(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(new(T), id).Error; err != nil {
		return err
	}

	r.EvictCache(ctx, id)

	return nil
}

func (r *resourceRepository[T]) Kill(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Unscoped().Delete(new(T), id).Error; err != nil {
		return err
	}

	r.EvictCache(ctx, id)

	return nil
}

func (r *resourceRepository[T]) Restore(ctx context.Context, id uuid.UUID) (*T, error) {
	if err := r.db.WithContext(ctx).
		Unscoped().
		Model(new(T)).
		Where("id = ?", id).
		Update("deleted_at", nil).
		Error; err != nil {
		return nil, err
	}

	r.EvictCache(ctx, id)

	return r.GetByID(ctx, id, nil)
}

func (r *resourceRepository[T]) GetMany(ctx context.Context, opts GetManyOptions) ([]T, int64, error) {
	opts.Normalize()

	var (
		entities []T
		total    int64
	)

	countQuery := r.db.WithContext(ctx).Model(new(T))
	countQuery = applyJoins(countQuery, opts.Joins)
	countQuery = applyFilters(countQuery, opts.Filters)
	countQuery = applyWheres(countQuery, opts.Wheres)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	listQuery := r.db.WithContext(ctx).Model(new(T))
	listQuery = applyJoins(listQuery, opts.Joins)
	listQuery = applyFilters(listQuery, opts.Filters)
	listQuery = applyWheres(listQuery, opts.Wheres)
	listQuery = applyPreloads(listQuery, opts.Preloads)
	if err := listQuery.Limit(opts.Limit).Offset(opts.Offset).Order(opts.OrderBy + " " + opts.OrderDirection).Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	return entities, total, nil
}

func applyFilters(db *gorm.DB, filters map[string]any) *gorm.DB {
	if len(filters) > 0 {
		return db.Where(filters)
	}
	return db
}

func applyWheres(db *gorm.DB, wheres []WhereClause) *gorm.DB {
	for _, where := range wheres {
		query := strings.TrimSpace(where.Query)
		if query == "" {
			continue
		}
		if len(where.Args) > 0 {
			db = db.Where(query, where.Args...)
			continue
		}
		db = db.Where(query)
	}
	return db
}

func applyJoins(db *gorm.DB, joins []JoinClause) *gorm.DB {
	for _, join := range joins {
		query := strings.TrimSpace(join.Query)
		if query == "" {
			continue
		}
		if len(join.Args) > 0 {
			db = db.Joins(query, join.Args...)
			continue
		}
		db = db.Joins(query)
	}
	return db
}

func applyPreloads(db *gorm.DB, preloads []string) *gorm.DB {
	for _, preload := range preloads {
		name := strings.TrimSpace(preload)
		if name == "" {
			continue
		}
		db = db.Preload(name)
	}
	return db
}

func (r *resourceRepository[T]) getCachedByID(ctx context.Context, id uuid.UUID) (*T, bool) {
	key := utils.GetModelCacheKey[T](id)
	var entity T

	if err := r.cache.GetJSON(ctx, key, &entity); err != nil {
		return nil, false
	}

	return &entity, true
}
