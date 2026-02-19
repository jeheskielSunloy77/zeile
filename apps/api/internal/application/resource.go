package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	"github.com/jeheskielSunloy77/zeile/internal/app/sqlerr"
	applicationdto "github.com/jeheskielSunloy77/zeile/internal/application/dto"
	"github.com/jeheskielSunloy77/zeile/internal/application/port"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"gorm.io/gorm"
)

type ResourceService[T domain.BaseModel, S applicationdto.StoreDTO[T], U applicationdto.UpdateDTO[T]] interface {
	Store(ctx context.Context, dto S) (*T, error)
	GetByID(ctx context.Context, id uuid.UUID, preloads []string) (*T, error)
	GetMany(ctx context.Context, opts port.GetManyOptions) ([]T, int64, error)
	Destroy(ctx context.Context, id uuid.UUID) error
	Kill(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, dto U) (*T, error)
	Restore(ctx context.Context, id uuid.UUID, preloads []string) (*T, error)
}

type resourceService[T domain.BaseModel, S applicationdto.StoreDTO[T], U applicationdto.UpdateDTO[T]] struct {
	repo         port.ResourceRepository[T]
	resourceName string
}

func NewResourceService[T domain.BaseModel, S applicationdto.StoreDTO[T], U applicationdto.UpdateDTO[T]](resourceName string, repo port.ResourceRepository[T]) ResourceService[T, S, U] {
	return &resourceService[T, S, U]{resourceName: resourceName, repo: repo}
}

func (s *resourceService[T, S, U]) Store(ctx context.Context, dto S) (*T, error) {
	entity := dto.ToModel()
	if err := s.repo.Store(ctx, entity); err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return entity, nil
}

func (s *resourceService[T, S, U]) GetByID(ctx context.Context, id uuid.UUID, preloads []string) (*T, error) {
	entity, err := s.repo.GetByID(ctx, id, preloads)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return entity, nil
}

func (s *resourceService[T, S, U]) GetMany(ctx context.Context, opts port.GetManyOptions) ([]T, int64, error) {
	entities, total, err := s.repo.GetMany(ctx, opts)
	if err != nil {
		return nil, 0, sqlerr.HandleError(err)
	}
	return entities, total, nil
}

func (s *resourceService[T, S, U]) Update(ctx context.Context, id uuid.UUID, dto U) (*T, error) {
	updates := dto.ToMap()

	entity, err := s.repo.GetByID(ctx, id, nil)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}

	if len(updates) == 0 {
		return entity, nil
	}

	updatedEntity, err := s.repo.Update(ctx, *entity, updates)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}

	return updatedEntity, nil
}

func (s *resourceService[T, S, U]) Destroy(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Destroy(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return errs.NewNotFoundError(s.resourceName+" not found", true)
		}
		return sqlerr.HandleError(err)
	}
	return nil
}

func (s *resourceService[T, S, U]) Kill(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Kill(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return errs.NewNotFoundError(s.resourceName+" not found", true)
		}
		return sqlerr.HandleError(err)
	}
	return nil
}

func (s *resourceService[T, S, U]) Restore(ctx context.Context, id uuid.UUID, preloads []string) (*T, error) {
	entity, err := s.repo.Restore(ctx, id)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}
	if len(preloads) == 0 {
		return entity, nil
	}

	entity, err = s.repo.GetByID(ctx, id, preloads)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return entity, nil
}
