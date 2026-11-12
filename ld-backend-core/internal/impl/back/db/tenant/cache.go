package ldtenant

import (
	"errors"
	"fmt"
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// CacheExpired tells if the cache expired, extending it by the duration when
// it did; a new cache is created already expired.
func (c *tenantsRepoImpl) CacheExpired(cacheName string, duration time.Duration) bool {
	expired := false
	err := c.transaction(func(tq *tenantQuery.Query) error {
		created, err := ensureCacheByName(tq, cacheName, duration)
		if err != nil {
			return fmt.Errorf("error ensuring cache by name: %w", err)
		}
		if created {
			expired = true
			return nil
		}

		update, err := tq.DbCacheTime.
			Where(tq.DbCacheTime.CacheTime.Lt(time.Now())).
			Where(tq.DbCacheTime.CacheName.Eq(cacheName)).
			Update(tq.DbCacheTime.CacheTime, time.Now().Add(duration))
		if err != nil {
			return fmt.Errorf("error updating cache times: %w", err)
		}
		expired = update.RowsAffected > 0
		return nil
	})
	if err != nil {
		slog.Error(fmt.Sprintf("error checking cache %s: %v", cacheName, err))
		return true
	}
	return expired
}

func (c *tenantsRepoImpl) UpdateCache(cacheName string, duration time.Duration) {
	err := c.transaction(func(tq *tenantQuery.Query) error {
		created, err := ensureCacheByName(tq, cacheName, duration)
		if err != nil {
			return fmt.Errorf("error ensuring cache by name: %w", err)
		}
		if created {
			return nil
		}

		_, err = tq.DbCacheTime.
			Where(tq.DbCacheTime.CacheName.Eq(cacheName)).
			Update(tq.DbCacheTime.CacheTime, time.Now().Add(duration))
		if err != nil {
			return fmt.Errorf("error updating cache times: %w", err)
		}
		return nil
	})
	if err != nil {
		slog.Error(fmt.Sprintf("error updating cache %s: %v", cacheName, err))
	}
}

func (c *tenantsRepoImpl) DeleteCache(cacheName string) {
	err := c.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbCacheTime.
			Unscoped().
			Where(tq.DbCacheTime.CacheName.Eq(cacheName)).
			Delete()
		return err
	})
	if err != nil {
		slog.Error(fmt.Sprintf("error deleting cache times for %s: %v", cacheName, err))
	}
}

// ensureCacheByName creates the cache when missing, telling if it did.
func ensureCacheByName(tq *tenantQuery.Query, cacheName string, duration time.Duration) (bool, error) {
	_, err := tq.DbCacheTime.Where(tq.DbCacheTime.CacheName.Eq(cacheName)).First()
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	err = tq.DbCacheTime.Save(&ldspec.DbCacheTime{
		CacheName: cacheName,
		CacheTime: time.Now().Add(duration),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *tenantsRepoImpl) FindNextExpirationTime(cacheName string) (time.Time, error) {
	tq := c.tenantDb
	cache, err := tq.DbCacheTime.
		Where(tq.DbCacheTime.CacheName.Eq(cacheName)).
		First()
	if err != nil {
		return time.Time{}, err
	}
	return cache.CacheTime, nil
}
