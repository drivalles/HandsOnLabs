package ldcore

import (
	ldconn "ld-backend/internal/impl/back/db/conn"
	ldspec "ld-backend/internal/spec"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// NewTestDBConfig is the configuration of a core database of the test's
// own, from the pool of core databases of the driver.
func NewTestDBConfig(t *testing.T, d ldconn.TestDriver) ldspec.DBConfig {
	t.Helper()
	return ldconn.NewTestDBConfig(t, d, "core")
}

// TestRepo is a core repo of the tests, with its connection to the
// database.
type TestRepo interface {
	ldspec.CoreRepo
	Db() *gorm.DB
}

// NewTestRepo creates a core repo on a database of the test's own.
func NewTestRepo(t *testing.T, d ldconn.TestDriver) TestRepo {
	t.Helper()
	return ConnectTestRepo(t, NewTestDBConfig(t, d))
}

// ConnectTestRepo creates a core repo on the database of the configuration,
// closing its connection when the test ends.
func ConnectTestRepo(t *testing.T, cfg ldspec.DBConfig) TestRepo {
	t.Helper()
	repoSpec, err := NewCoreRepo(cfg)
	require.NoError(t, err)
	repo := repoSpec.(*coreRepoImpl)
	t.Cleanup(func() {
		ldconn.CloseDb(repo.coreDb)
	})
	return repo
}
