package ldtenant

import (
	"errors"
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"
	"sync"

	"gorm.io/gorm"
)

type tenantsManager struct {
	core            ldspec.CoreRepo
	tenantDbManager *tenantDbManager
	// mu guards tenantsMap and locks; it is not held while connecting, so a
	// tenant being created does not hold up the others.
	mu         sync.RWMutex
	tenantsMap map[string]ldspec.TenantRepo
	// locks lets one call at a time connect each location, so the calls
	// arriving together for a new tenant do not create it more than once.
	locks map[string]*sync.Mutex
}

func NewTenantsManager(core ldspec.CoreRepo, config ldspec.DBConfig) ldspec.TenantsManager {
	return &tenantsManager{
		core:            core,
		tenantDbManager: newTenantDbManager(core, config),
		tenantsMap:      map[string]ldspec.TenantRepo{},
		locks:           map[string]*sync.Mutex{},
	}
}

func (t *tenantsManager) FindOrCreateTenant(locationId string) (ldspec.TenantRepo, error) {
	if tenant, ok := t.cached(locationId); ok {
		return tenant, nil
	}
	defer t.lockLocation(locationId)()
	if tenant, ok := t.cached(locationId); ok {
		return tenant, nil
	}
	db, err := t.tenantDbManager.findOrCreate(locationId)
	if err != nil {
		return nil, err
	}
	if db != nil {
		return t.cache(locationId, db), nil
	}
	return nil, nil
}

func (t *tenantsManager) GetTenant(locationId string) (ldspec.TenantRepo, error) {
	if tenant, ok := t.cached(locationId); ok {
		return tenant, nil
	}
	defer t.lockLocation(locationId)()
	if tenant, ok := t.cached(locationId); ok {
		return tenant, nil
	}
	db, err := t.tenantDbManager.find(locationId)
	if err != nil {
		return nil, err
	}
	if db != nil {
		return t.cache(locationId, db), nil
	}
	return nil, errors.New("tenant not found")
}

func (t *tenantsManager) cached(locationId string) (ldspec.TenantRepo, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	tenant, ok := t.tenantsMap[locationId]
	return tenant, ok
}

// cache keeps the repo of the tenant connected to the database.
func (t *tenantsManager) cache(locationId string, db *gorm.DB) ldspec.TenantRepo {
	t.mu.Lock()
	defer t.mu.Unlock()
	tenant := newTenantsRepo(db, tenantQuery.Use(db))
	t.tenantsMap[locationId] = tenant
	return tenant
}

// lockLocation waits until no other call is connecting the location, and
// returns the function releasing it.
func (t *tenantsManager) lockLocation(locationId string) func() {
	t.mu.Lock()
	lock, ok := t.locks[locationId]
	if !ok {
		lock = &sync.Mutex{}
		t.locks[locationId] = lock
	}
	t.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}
