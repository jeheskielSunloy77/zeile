package application

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/repository"

	"github.com/stretchr/testify/require"
)

type testEntity struct {
	ID   uuid.UUID
	Name string
}

func (m testEntity) GetID() uuid.UUID {
	return m.ID
}

type testStoreDTO struct {
	Name string
}

func (d testStoreDTO) Validate() error { return nil }

func (d testStoreDTO) ToModel() *testEntity {
	return &testEntity{Name: d.Name}
}

type testUpdateDTO struct {
	Name *string
}

func (d testUpdateDTO) Validate() error { return nil }

func (d testUpdateDTO) ToModel() *testEntity { return &testEntity{} }

func (d testUpdateDTO) ToMap() map[string]any {
	updates := make(map[string]any)
	if d.Name != nil {
		updates["name"] = *d.Name
	}
	return updates
}

type trackingResourceRepo struct {
	*repository.MockResourceRepository[testEntity]
	updateCalled *bool
}

func (m *trackingResourceRepo) Update(ctx context.Context, entity testEntity, updates ...map[string]any) (*testEntity, error) {
	if m.updateCalled != nil {
		*m.updateCalled = true
	}
	return m.MockResourceRepository.Update(ctx, entity, updates...)
}

// Ensures GetByID maps not-found repository errors to HTTP 404 responses.
func TestResourceServiceGetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMockResourceRepository[testEntity](false)

	svc := NewResourceService[testEntity, testStoreDTO, testUpdateDTO]("widget", repo)

	_, err := svc.GetByID(ctx, uuid.New(), nil)
	require.Error(t, err)

	var httpErr *errs.ErrorResponse
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusNotFound, httpErr.Status)
}

// Ensures Update returns the existing entity when there are no updates.
func TestResourceServiceUpdate_NoUpdates(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	updateCalled := false

	repo := &trackingResourceRepo{
		MockResourceRepository: repository.NewMockResourceRepository[testEntity](false),
		updateCalled:           &updateCalled,
	}
	require.NoError(t, repo.Store(ctx, &testEntity{ID: id, Name: "current"}))

	svc := NewResourceService[testEntity, testStoreDTO, testUpdateDTO]("widget", repo)

	updated, err := svc.Update(ctx, id, testUpdateDTO{})
	require.NoError(t, err)
	require.False(t, updateCalled)
	require.NotNil(t, updated)
	require.Equal(t, "current", updated.Name)
}
