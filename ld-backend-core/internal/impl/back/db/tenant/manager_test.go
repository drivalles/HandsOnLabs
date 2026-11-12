package ldtenant_test

import (
	"errors"
	"fmt"
	ldconn "ld-backend/internal/impl/back/db/conn"
	ldcore "ld-backend/internal/impl/back/db/core"
	ldtenant "ld-backend/internal/impl/back/db/tenant"
	ldutils "ld-backend/internal/impl/core/utils"
	ldspec "ld-backend/internal/spec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// stubCoreRepo delegates to a core repo and can fail FindTenant and
// SaveTenant.
type stubCoreRepo struct {
	ldspec.CoreRepo
	findErr   error
	saveErr   error
	saveCalls atomic.Int32
}

func (s *stubCoreRepo) FindTenant(id string) (*ldspec.DbTenant, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	return s.CoreRepo.FindTenant(id)
}

func (s *stubCoreRepo) SaveTenant(tenant *ldspec.DbTenant) error {
	s.saveCalls.Add(1)
	if s.saveErr != nil {
		return s.saveErr
	}
	return s.CoreRepo.SaveTenant(tenant)
}

// managerTest is a tenants manager on a core database of the test's own,
// creating the tenant databases from the url template of the driver.
type managerTest struct {
	d       ldconn.TestDriver
	cfg     ldspec.DBConfig
	core    *stubCoreRepo
	manager ldspec.TenantsManager
}

func newManagerTest(t *testing.T, d ldconn.TestDriver) *managerTest {
	t.Helper()
	cfg := ldconn.NewTestDBTemplate(t, d)
	core := &stubCoreRepo{CoreRepo: ldcore.NewTestRepo(t, d)}
	return &managerTest{d: d, cfg: cfg, core: core, manager: ldtenant.NewTenantsManager(core, cfg)}
}

// newManager is another manager of the same core database and template, as
// on another server node.
func (m *managerTest) newManager() ldspec.TenantsManager {
	return ldtenant.NewTenantsManager(m.core, m.cfg)
}

// businessId is a business id of the test's own, so its tenant database is
// not used by other tests or runs; the database the manager creates for it
// on a server is dropped when the test ends.
func (m *managerTest) businessId(t *testing.T, prefix string) string {
	t.Helper()
	id := prefix + "_" + strings.ToLower(ldutils.GenerateRandomString(8))
	ldconn.DropTestDatabase(t, m.d, m.cfg, ldtenant.TestTenantDbName(id))
	return id
}

// connected returns the tenant repo, closing its connection when the test
// ends.
func connected(t *testing.T, repo ldspec.TenantRepo, err error) ldtenant.TestRepo {
	t.Helper()
	require.NoError(t, err)
	require.NotNil(t, repo)
	testRepo, ok := repo.(ldtenant.TestRepo)
	require.True(t, ok)
	t.Cleanup(func() {
		ldconn.CloseDb(testRepo.Db())
	})
	return testRepo
}

// findOrCreate finds or creates the tenant of the business with the
// manager, closing its connection when the test ends.
func findOrCreate(t *testing.T, manager ldspec.TenantsManager, businessId string) ldtenant.TestRepo {
	t.Helper()
	repo, err := manager.FindOrCreateTenant(businessId)
	return connected(t, repo, err)
}

// getTenant gets the existing tenant of the business with the manager,
// closing its connection when the test ends.
func getTenant(t *testing.T, manager ldspec.TenantsManager, businessId string) ldtenant.TestRepo {
	t.Helper()
	repo, err := manager.GetTenant(businessId)
	return connected(t, repo, err)
}

// TestTenantsManager runs all the cases of the tenants manager with each
// database driver. Creating a tenant database takes long, above all on
// cockroachdb, so the cases share the tenants they create where they can.
func TestTenantsManager(t *testing.T) {
	for _, d := range ldconn.TestDrivers() {
		t.Run(d.Name, func(t *testing.T) {
			d.Check(t)

			t.Run("CreatedTenant", func(t *testing.T) {
				testCreatedTenant(t, d)
			})

			t.Run("FindOrCreateTenantErrors", func(t *testing.T) {
				testFindOrCreateTenantErrors(t, d)
			})

			t.Run("GetTenantErrors", func(t *testing.T) {
				testGetTenantErrors(t, d)
			})
		})
	}
}

// testCreatedTenant creates the tenant of a new business, asked from several
// workers at once, and checks how the managers use it.
func testCreatedTenant(t *testing.T, d ldconn.TestDriver) {
	m := newManagerTest(t, d)
	id := m.businessId(t, "Loc_ABC")
	const workers = 4
	repos := make([]ldspec.TenantRepo, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			repos[i], errs[i] = m.manager.FindOrCreateTenant(id)
		}()
	}
	wg.Wait()
	repo := connected(t, repos[0], errs[0])
	saveUser(t, repo, "user-1", "User 1")

	t.Run("creates the tenant once for the concurrent calls", func(t *testing.T) {
		for i := 1; i < workers; i++ {
			require.NoError(t, errs[i])
			assert.Same(t, repo, repos[i].(ldtenant.TestRepo))
		}
		assert.EqualValues(t, 1, m.core.saveCalls.Load())
	})

	t.Run("saves the tenant in the core database", func(t *testing.T) {
		tenant, err := m.core.FindTenant(id)
		require.NoError(t, err)
		dbName := ldtenant.TestTenantDbName(id)
		assert.Equal(t, "loc_"+strings.ToLower(id), dbName)
		assert.Equal(t, id, tenant.BusinessId)
		assert.Equal(t, id, tenant.Title)
		assert.Equal(t, dbName, tenant.DbName)
		assert.Equal(t, fmt.Sprintf(m.cfg.DbUrlTpl, dbName), tenant.Url)
		assert.Equal(t, string(m.cfg.Driver), tenant.Driver)
		assert.Equal(t, ldspec.LDActive, tenant.Status)
	})

	t.Run("creates and migrates the tenant database", func(t *testing.T) {
		if d.IsServer() {
			assert.True(t, ldconn.TestDatabaseExists(t, d, m.cfg, ldtenant.TestTenantDbName(id)))
		}
		migrator := repo.Db().Migrator()
		for _, table := range ldspec.DbTenantTables {
			assert.True(t, migrator.HasTable(table), "missing table for %T", table)
		}
	})

	t.Run("returns the same repo to later calls", func(t *testing.T) {
		again, err := m.manager.FindOrCreateTenant(id)
		require.NoError(t, err)
		assert.Same(t, repo, again)
		found, err := m.manager.GetTenant(id)
		require.NoError(t, err)
		assert.Same(t, repo, found)
		assert.EqualValues(t, 1, m.core.saveCalls.Load())
	})

	t.Run("another manager finds the existing tenant", func(t *testing.T) {
		other := findOrCreate(t, m.newManager(), id)
		assert.NotSame(t, repo, other)
		_, err := other.FindUser(ldutils.Ptr("user-1"))
		assert.NoError(t, err, "the database of the tenant is opened")
		assert.EqualValues(t, 1, m.core.saveCalls.Load(), "the tenant is saved only when created")
	})

	t.Run("another manager gets the existing tenant", func(t *testing.T) {
		other := getTenant(t, m.newManager(), id)
		_, err := other.FindUser(ldutils.Ptr("user-1"))
		assert.NoError(t, err, "the database of the tenant is opened")
	})

	t.Run("each business has its own database", func(t *testing.T) {
		other := findOrCreate(t, m.manager, m.businessId(t, "loc2"))
		users, err := other.FindUsers()
		require.NoError(t, err)
		assert.Empty(t, users)
		assert.EqualValues(t, 2, m.core.saveCalls.Load())
	})
}

func testFindOrCreateTenantErrors(t *testing.T, d ldconn.TestDriver) {
	t.Run("inactive tenant", func(t *testing.T) {
		m := newManagerTest(t, d)
		id := m.businessId(t, "loc1")
		require.NoError(t, m.core.SaveTenant(&ldspec.DbTenant{BusinessId: id, Driver: string(m.cfg.Driver), Status: ldspec.LDInactive}))

		repo, err := m.manager.FindOrCreateTenant(id)
		assert.ErrorContains(t, err, fmt.Sprintf("tenant %s is not active", id))
		assert.Nil(t, repo)
	})

	t.Run("core error finding the tenant", func(t *testing.T) {
		m := newManagerTest(t, d)
		m.core.findErr = errors.New("core is down")

		repo, err := m.manager.FindOrCreateTenant(m.businessId(t, "loc1"))
		assert.ErrorIs(t, err, m.core.findErr)
		assert.Nil(t, repo)
		assert.Zero(t, m.core.saveCalls.Load(), "no tenant is created on a core error")
	})

	t.Run("core error saving the new tenant", func(t *testing.T) {
		m := newManagerTest(t, d)
		m.core.saveErr = errors.New("save failed")

		repo, err := m.manager.FindOrCreateTenant(m.businessId(t, "loc1"))
		assert.ErrorIs(t, err, m.core.saveErr)
		assert.ErrorContains(t, err, "could not find or create tenant")
		assert.Nil(t, repo)
	})

	t.Run("unsupported driver", func(t *testing.T) {
		m := newManagerTest(t, d)
		cfg := m.cfg
		cfg.Driver = "mysql"
		manager := ldtenant.NewTenantsManager(m.core, cfg)

		repo, err := manager.FindOrCreateTenant("loc1")
		assert.ErrorContains(t, err, "unsupported driver mysql")
		assert.Nil(t, repo)
		_, err = m.core.FindTenant("loc1")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "the tenant is not saved")
	})

	t.Run("stored tenant with an unsupported driver", func(t *testing.T) {
		m := newManagerTest(t, d)
		require.NoError(t, m.core.SaveTenant(&ldspec.DbTenant{BusinessId: "loc1", Driver: "mysql", Url: "some-url", Status: ldspec.LDActive}))

		repo, err := m.manager.FindOrCreateTenant("loc1")
		assert.ErrorContains(t, err, "unsupported driver mysql")
		assert.Nil(t, repo)
	})

	t.Run("server not reachable", func(t *testing.T) {
		if !d.IsServer() {
			t.Skip("sqlite has no server")
		}
		m := newManagerTest(t, d)
		cfg := m.cfg
		cfg.DbUrlTpl = ldconn.UnreachableUrlTpl(t)
		manager := ldtenant.NewTenantsManager(m.core, cfg)

		repo, err := manager.FindOrCreateTenant("loc1")
		assert.ErrorContains(t, err, "could not find or create tenant")
		assert.Nil(t, repo)
		_, err = m.core.FindTenant("loc1")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "the tenant is not saved")
	})
}

func testGetTenantErrors(t *testing.T, d ldconn.TestDriver) {
	t.Run("a missing tenant is not created", func(t *testing.T) {
		m := newManagerTest(t, d)
		id := m.businessId(t, "loc1")

		repo, err := m.manager.GetTenant(id)
		assert.Error(t, err)
		assert.Nil(t, repo)
		assert.Zero(t, m.core.saveCalls.Load())
		if d.IsServer() {
			assert.False(t, ldconn.TestDatabaseExists(t, d, m.cfg, ldtenant.TestTenantDbName(id)))
		}
	})

	t.Run("inactive tenant", func(t *testing.T) {
		m := newManagerTest(t, d)
		id := m.businessId(t, "loc1")
		require.NoError(t, m.core.SaveTenant(&ldspec.DbTenant{BusinessId: id, Driver: string(m.cfg.Driver), Status: ldspec.LDInactive}))

		repo, err := m.manager.GetTenant(id)
		assert.ErrorContains(t, err, fmt.Sprintf("tenant %s is not active", id))
		assert.Nil(t, repo)
	})

	t.Run("core error", func(t *testing.T) {
		m := newManagerTest(t, d)
		m.core.findErr = errors.New("core is down")

		repo, err := m.manager.GetTenant("loc1")
		assert.ErrorIs(t, err, m.core.findErr)
		assert.Nil(t, repo)
	})
}
