package ldtenant

import (
	"fmt"
	ldconn "ld-backend/internal/impl/back/db/conn"
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	ldspec "ld-backend/internal/spec"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestRepo is a tenant repo of the tests, with its connection to the
// database.
type TestRepo interface {
	ldspec.TenantRepo
	Db() *gorm.DB
}

// Db is the connection of the repo to the tenant database.
func (c *tenantsRepoImpl) Db() *gorm.DB {
	return c.db
}

// NewTestRepo creates a tenant repo on a database of the test's own, from
// the pool of tenant databases of the driver, closing its connection when
// the test ends.
func NewTestRepo(t *testing.T, d ldconn.TestDriver) TestRepo {
	t.Helper()
	cfg := ldconn.NewTestDBConfig(t, d, "tenant")
	dialector, err := ldconn.FindDialector(cfg.Driver, fmt.Sprintf(cfg.DbUrlTpl, cfg.CoreDBName))
	require.NoError(t, err)
	db, err := connect(dialector)
	require.NoError(t, err)
	t.Cleanup(func() {
		closeDb(db)
	})
	return newTenantsRepo(db, tenantQuery.Use(db)).(*tenantsRepoImpl)
}

// TestTenantDbName is the name of the database the manager creates for the
// business.
func TestTenantDbName(businessId string) string {
	return tenantDbName(businessId)
}
