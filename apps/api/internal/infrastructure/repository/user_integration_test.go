package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/cache"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/utils"
	internaltesting "github.com/jeheskielSunloy77/zeile/internal/testing"
	"gorm.io/gorm"

	"github.com/stretchr/testify/require"
)

// Ensures the user repository supports CRUD operations and pagination against Postgres.
func TestUserRepository_ResourceLifecycle(t *testing.T) {
	testDB, cleanup := internaltesting.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := internaltesting.WithRollbackTransaction(ctx, testDB, func(tx *gorm.DB) error {
		repo := NewUserRepository(&config.Config{}, tx, nil)

		user1 := &domain.User{ID: uuid.New(), Email: "user1@example.com", Username: "user1"}
		user2 := &domain.User{ID: uuid.New(), Email: "user2@example.com", Username: "user2"}

		require.NoError(t, repo.Store(ctx, user1))
		require.NoError(t, repo.Store(ctx, user2))
		require.NotEqual(t, uuid.Nil, user1.ID)

		fetched, err := repo.GetByID(ctx, user1.ID, nil)
		require.NoError(t, err)
		require.Equal(t, user1.ID, fetched.ID)

		updates := map[string]any{"username": "user1-updated"}
		_, err = repo.Update(ctx, *fetched, updates)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, user1.ID, nil)
		require.NoError(t, err)
		require.Equal(t, "user1-updated", updated.Username)

		list, total, err := repo.GetMany(ctx, GetManyOptions{Limit: 1, Offset: 0, OrderBy: "created_at", OrderDirection: "asc"})
		require.NoError(t, err)
		require.Equal(t, int64(2), total)
		require.Len(t, list, 1)

		require.NoError(t, repo.Destroy(ctx, user1.ID))
		_, err = repo.GetByID(ctx, user1.ID, nil)
		require.Error(t, err)

		restored, err := repo.Restore(ctx, user1.ID)
		require.NoError(t, err)
		require.Equal(t, user1.ID, restored.ID)

		return nil
	})
	require.NoError(t, err)
}

type testCache struct {
	values  map[string][]byte
	deletes []string
}

func newTestCache() *testCache {
	return &testCache{
		values: make(map[string][]byte),
	}
}

func (c *testCache) Get(ctx context.Context, key string) ([]byte, error) {
	value, ok := c.values[key]
	if !ok {
		return nil, cache.ErrCacheMiss
	}
	return value, nil
}

func (c *testCache) Set(ctx context.Context, key string, value []byte, ttl ...time.Duration) error {
	c.values[key] = append([]byte(nil), value...)
	return nil
}

func (c *testCache) SetJSON(ctx context.Context, key string, value any, ttl ...time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.values[key] = data
	return nil
}

func (c *testCache) GetJSON(ctx context.Context, key string, dest any) error {
	data, ok := c.values[key]
	if !ok {
		return cache.ErrCacheMiss
	}
	if err := json.Unmarshal(data, dest); err != nil {
		_ = c.Delete(ctx, key)
		return cache.ErrCacheMiss
	}
	return nil
}

func (c *testCache) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		delete(c.values, key)
		c.deletes = append(c.deletes, key)
	}
	return nil
}

func TestUserRepository_CacheLifecycle(t *testing.T) {
	testDB, cleanup := internaltesting.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := internaltesting.WithRollbackTransaction(ctx, testDB, func(tx *gorm.DB) error {
		cacheClient := newTestCache()
		repo := NewUserRepository(&config.Config{
			Cache: config.CacheConfig{
				TTL: 5 * time.Minute,
			},
		}, tx, cacheClient)

		user := &domain.User{ID: uuid.New(), Email: "cache1@example.com", Username: "cache1"}
		require.NoError(t, repo.Store(ctx, user))

		key := "resource:" + utils.GetModelNameLower[domain.User]() + ":id:" + user.ID.String()
		_, ok := cacheClient.values[key]
		require.True(t, ok, "expected cache entry after store")

		require.NoError(t, tx.Delete(&domain.User{}, user.ID).Error)

		cached, err := repo.GetByID(ctx, user.ID, nil)
		require.NoError(t, err)
		require.Equal(t, user.ID, cached.ID)

		user2 := &domain.User{ID: uuid.New(), Email: "cache2@example.com", Username: "cache2"}
		require.NoError(t, repo.Store(ctx, user2))

		key2 := "resource:" + utils.GetModelNameLower[domain.User]() + ":id:" + user2.ID.String()
		require.NotEmpty(t, cacheClient.values[key2])

		fetched, err := repo.GetByID(ctx, user2.ID, nil)
		require.NoError(t, err)

		_, err = repo.Update(ctx, *fetched, map[string]any{"username": "cache2-updated"})
		require.NoError(t, err)
		_, ok = cacheClient.values[key2]
		require.False(t, ok, "expected cache eviction on update")

		user3 := &domain.User{ID: uuid.New(), Email: "cache3@example.com", Username: "cache3"}
		require.NoError(t, repo.Store(ctx, user3))

		key3 := "resource:" + utils.GetModelNameLower[domain.User]() + ":id:" + user3.ID.String()
		require.NotEmpty(t, cacheClient.values[key3])

		require.NoError(t, repo.Destroy(ctx, user3.ID))
		_, ok = cacheClient.values[key3]
		require.False(t, ok, "expected cache eviction on destroy")

		return nil
	})
	require.NoError(t, err)
}
