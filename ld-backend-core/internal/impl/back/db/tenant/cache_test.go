package ldtenant_test

import (
	ldconn "ld-backend/internal/impl/back/db/conn"
	ldtenant "ld-backend/internal/impl/back/db/tenant"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testCacheExpired(t *testing.T, d ldconn.TestDriver) {
	t.Run("a new cache is expired and created for the duration", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		assert.True(t, repo.CacheExpired("pipelines", time.Hour))

		next, err := repo.FindNextExpirationTime("pipelines")
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().Add(time.Hour), next, time.Minute)
	})

	t.Run("a cache within its duration is not expired", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		require.True(t, repo.CacheExpired("pipelines", time.Hour))
		assert.False(t, repo.CacheExpired("pipelines", time.Hour))
	})

	t.Run("an expired cache is renewed for the duration", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		repo.UpdateCache("pipelines", -time.Minute)

		assert.True(t, repo.CacheExpired("pipelines", time.Hour))
		assert.False(t, repo.CacheExpired("pipelines", time.Hour), "the first caller renews it")
		next, err := repo.FindNextExpirationTime("pipelines")
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().Add(time.Hour), next, time.Minute)
	})

	t.Run("the caches are kept apart by name", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		require.True(t, repo.CacheExpired("pipelines", time.Hour))
		assert.True(t, repo.CacheExpired("users", time.Hour))
		assert.False(t, repo.CacheExpired("pipelines", time.Hour))
	})
}

func testUpdateCache(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates a missing cache", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		repo.UpdateCache("pipelines", time.Hour)

		next, err := repo.FindNextExpirationTime("pipelines")
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().Add(time.Hour), next, time.Minute)
		assert.False(t, repo.CacheExpired("pipelines", time.Hour))
	})

	t.Run("sets the expiration of an existing cache", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		repo.UpdateCache("pipelines", time.Hour)
		repo.UpdateCache("pipelines", 3*time.Hour)

		next, err := repo.FindNextExpirationTime("pipelines")
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().Add(3*time.Hour), next, time.Minute)
	})

	t.Run("a negative duration expires the cache", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		repo.UpdateCache("pipelines", time.Hour)
		repo.UpdateCache("pipelines", -time.Minute)
		assert.True(t, repo.CacheExpired("pipelines", time.Hour))
	})
}

func testDeleteCache(t *testing.T, d ldconn.TestDriver) {
	t.Run("a deleted cache is expired", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		repo.UpdateCache("pipelines", time.Hour)
		repo.UpdateCache("users", time.Hour)

		repo.DeleteCache("pipelines")
		_, err := repo.FindNextExpirationTime("pipelines")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.True(t, repo.CacheExpired("pipelines", time.Hour))
		assert.False(t, repo.CacheExpired("users", time.Hour), "other caches are kept")
	})

	t.Run("a missing cache is ignored", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		repo.DeleteCache("pipelines")
		_, err := repo.FindNextExpirationTime("pipelines")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func testFindNextExpirationTime(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	repo.UpdateCache("pipelines", time.Hour)
	repo.UpdateCache("users", 2*time.Hour)

	t.Run("expiration of the cache", func(t *testing.T) {
		next, err := repo.FindNextExpirationTime("users")
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().Add(2*time.Hour), next, time.Minute)
	})

	t.Run("unknown cache", func(t *testing.T) {
		next, err := repo.FindNextExpirationTime("unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.True(t, next.IsZero())
	})
}
