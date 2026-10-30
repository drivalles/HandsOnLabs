package core

import (
	"fmt"
	"ld-backend/internal/impl/utils"
	"ld-backend/internal/spec"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var cfgServ = utils.NewConfigService()

// TestDriver is a database engine the repo tests run against.
type TestDriver struct {
	Name string
	Cfg  spec.DBTestConfig
	// AdminDb is the database used to create and drop the test databases on
	// a server; empty for SQLite.
	AdminDb string
	// DropSql drops the test database with the name.
	DropSql string
}

// TestDrivers are all the database engines of the tests.
func TestDrivers() []TestDriver {
	return []TestDriver{
		{
			Name: "sqlite-memory",
			Cfg:  cfgServ.InMemoryDBConfig(),
		},
		{
			Name: "sqlite-file",
			Cfg:  cfgServ.TestdataDBFolderConfig(),
		},
		{
			Name:    "postgres",
			Cfg:     cfgServ.PostgresDBConfig(spec.TestNetworkInternal),
			AdminDb: "postgres",
			DropSql: `DROP DATABASE IF EXISTS "%s" WITH (FORCE)`,
		},
		{
			Name:    "cockroachdb",
			Cfg:     cfgServ.CockroachDBConfig(spec.TestNetworkInternal),
			AdminDb: "defaultdb",
			DropSql: `DROP DATABASE IF EXISTS "%s" CASCADE`,
		},
	}
}

// Check skips the driver when none of its servers is up.
func (d TestDriver) Check(t *testing.T) {
	t.Helper()
	if d.AdminDb == "" {
		return
	}
	for _, host := range strings.Split(d.Cfg.PostgresTestIp, ",") {
		addr := net.JoinHostPort(host, fmt.Sprint(d.Cfg.PostgresTestPort))
		if conn, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
			_ = conn.Close()
			return
		}
	}
	t.Skipf("%s server is not reachable at %s", d.Name, d.Cfg.PostgresTestIp)
}

// RunTestsWithDbs runs the tests of the package and drops the test databases
// created on the servers; call it from TestMain.
func RunTestsWithDbs(m *testing.M) {
	code := m.Run()
	testDbs.dropAll()
	os.Exit(code)
}

// NewTestDBConfig is the configuration of a database of the test's own, so
// the tests, and the subtests within them, do not see each other's records.
// SQLite gets a new database each time; the servers reuse emptied databases
// from a pool, as migrating a new one takes too long, mostly on cockroachdb.
// Close the connections to the database before the test ends.
func NewTestDBConfig(t *testing.T, d TestDriver) spec.DBConfig {
	t.Helper()
	testCfg := d.Cfg
	if testCfg.Driver == spec.SqliteDriver && !testCfg.InMemorySQLite {
		testCfg.SQLiteDir = t.TempDir()
	}
	cfg := testCfg.ToDBConfig()
	if d.AdminDb == "" {
		cfg.CoreDBName = newTestDbName()
		return cfg
	}

	cfg.CoreDBName = testDbs.take(t, d, cfg)
	// registered before the test opens the database, so it runs after the
	// test closes it
	t.Cleanup(func() {
		if clearTables(t, cfg) {
			testDbs.release(d, cfg.CoreDBName)
		}
	})
	return cfg
}

// NewTestRepo creates a core repo on a database of the test's own.
func NewTestRepo(t *testing.T, d TestDriver) spec.CoreRepo {
	t.Helper()
	return newTestRepo(t, d)
}

func newTestRepo(t *testing.T, d TestDriver) *coreRepoImpl {
	t.Helper()
	repoSpec, err := NewCoreRepo(NewTestDBConfig(t, d))
	require.NoError(t, err)
	repo := repoSpec.(*coreRepoImpl)
	t.Cleanup(func() {
		closeDb(repo.coreDb)
	})
	return repo
}

// CompanyGrant is a company install, without a location, created at the time.
func CompanyGrant(companyId string, authCode string, createdAt time.Time) *spec.DbGHLAppInstallation {
	return &spec.DbGHLAppInstallation{
		Model:             gorm.Model{CreatedAt: createdAt},
		AuthorizationCode: authCode,
		CompanyId:         utils.PtrString(companyId),
		AccessToken:       utils.PtrString("company-token-" + authCode),
	}
}

// LocationAccess is a location token of the company created at the time.
func LocationAccess(companyId string, locationId string, authCode string, createdAt time.Time) *spec.DbGHLAppInstallation {
	return &spec.DbGHLAppInstallation{
		Model:             gorm.Model{CreatedAt: createdAt},
		AuthorizationCode: authCode,
		CompanyId:         utils.PtrString(companyId),
		LocationId:        utils.PtrString(locationId),
		AccessToken:       utils.PtrString("location-token-" + authCode),
	}
}

// SaveInstalls saves the installations in the repo.
func SaveInstalls(t *testing.T, repo spec.CoreRepo, installs ...*spec.DbGHLAppInstallation) {
	t.Helper()
	for _, install := range installs {
		require.NoError(t, repo.SaveAppInstall(install))
	}
}

func newTestDbName() string {
	return strings.ToLower("ldcore_test_" + utils.GenerateRandomString(12))
}

// clearTables deletes the records of all the tables, so the database can be
// used by another test; it returns false when it could not.
func clearTables(t *testing.T, cfg spec.DBConfig) bool {
	db, err := openDb(cfg.Driver, fmt.Sprintf(cfg.DbUrlTpl, cfg.CoreDBName))
	if err != nil {
		t.Logf("could not connect to clear the test database: %v", err)
		return false
	}
	defer closeDb(db)
	tables, err := db.Migrator().GetTables()
	if err != nil {
		t.Logf("could not list the tables to clear: %v", err)
		return false
	}
	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf(`DELETE FROM "%s"`, table)).Error; err != nil {
			t.Logf("could not clear table %s: %v", table, err)
			return false
		}
	}
	return true
}

// testDbs are the test databases created on the servers.
var testDbs = &testDbPool{free: map[string][]string{}, created: map[string]*poolDriver{}}

type testDbPool struct {
	mu      sync.Mutex
	free    map[string][]string
	created map[string]*poolDriver
}

// poolDriver is what is needed to drop the databases created on a server.
type poolDriver struct {
	driver TestDriver
	cfg    spec.DBConfig
	names  []string
}

// take returns an empty database of the driver, creating it when none is
// free.
func (p *testDbPool) take(t *testing.T, d TestDriver, cfg spec.DBConfig) string {
	t.Helper()
	p.mu.Lock()
	if free := p.free[d.Name]; len(free) > 0 {
		name := free[len(free)-1]
		p.free[d.Name] = free[:len(free)-1]
		p.mu.Unlock()
		return name
	}
	p.mu.Unlock()

	name := newTestDbName()
	admin, err := openDb(cfg.Driver, fmt.Sprintf(cfg.DbUrlTpl, d.AdminDb))
	require.NoError(t, err)
	defer closeDb(admin)
	require.NoError(t, admin.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, name)).Error)

	p.mu.Lock()
	defer p.mu.Unlock()
	created, ok := p.created[d.Name]
	if !ok {
		created = &poolDriver{driver: d, cfg: cfg}
		p.created[d.Name] = created
	}
	created.names = append(created.names, name)
	return name
}

func (p *testDbPool) release(d TestDriver, name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.free[d.Name] = append(p.free[d.Name], name)
}

// dropAll drops all the databases created on the servers.
func (p *testDbPool) dropAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, created := range p.created {
		admin, err := openDb(created.cfg.Driver, fmt.Sprintf(created.cfg.DbUrlTpl, created.driver.AdminDb))
		if err != nil {
			fmt.Printf("could not connect to drop the %s test databases: %v\n", created.driver.Name, err)
			continue
		}
		for _, name := range created.names {
			if err := admin.Exec(fmt.Sprintf(created.driver.DropSql, name)).Error; err != nil {
				fmt.Printf("could not drop test database %s: %v\n", name, err)
			}
		}
		closeDb(admin)
	}
	p.created = map[string]*poolDriver{}
	p.free = map[string][]string{}
}

// openDb connects to the database without migrating it.
func openDb(driver spec.LDDatabaseDriver, url string) (*gorm.DB, error) {
	dialector, err := findDialector(driver, url)
	if err != nil {
		return nil, err
	}
	return gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func closeDb(db *gorm.DB) {
	if sqlDb, err := db.DB(); err == nil {
		_ = sqlDb.Close()
	}
}
