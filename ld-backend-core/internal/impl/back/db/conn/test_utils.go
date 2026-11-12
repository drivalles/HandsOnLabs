package ldconn

import (
	"fmt"
	ldcfg "ld-backend/internal/impl/core/cfg"
	ldutils "ld-backend/internal/impl/core/utils"
	ldspec "ld-backend/internal/spec"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var cfgServ = ldcfg.NewConfigService()

// TestDriver is a database engine the repo tests run against.
type TestDriver struct {
	Name string
	Cfg  ldspec.DBTestConfig
	// AdminDb is the database used to create and drop the test databases on
	// a server; empty for SQLite.
	AdminDb string
	// DropSql drops the test database with the name.
	DropSql string
}

func GetTestDriver(name string) TestDriver {
	for _, d := range TestDrivers() {
		if d.Name == name {
			return d
		}
	}
	panic(fmt.Sprintf("unknown test driver %s", name))
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
			Cfg:     cfgServ.PostgresDBConfig(ldspec.TestNetworkInternal),
			AdminDb: "postgres",
			DropSql: `DROP DATABASE IF EXISTS "%s" WITH (FORCE)`,
		},
		{
			Name:    "cockroachdb",
			Cfg:     cfgServ.CockroachDBConfig(ldspec.TestNetworkInternal),
			AdminDb: "defaultdb",
			DropSql: `DROP DATABASE IF EXISTS "%s" CASCADE`,
		},
	}
}

// Check skips the driver when none of its servers is up.
func (d TestDriver) Check(t *testing.T) {
	t.Helper()
	if !d.IsServer() {
		return
	}
	for _, host := range strings.Split(d.Cfg.TestServerIp, ",") {
		addr := net.JoinHostPort(host, fmt.Sprint(d.Cfg.TestServerPort))
		if conn, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
			_ = conn.Close()
			return
		}
	}
	t.Skipf("%s server is not reachable at %s", d.Name, d.Cfg.TestServerIp)
}

// IsServer tells if the driver connects to a database server instead of
// SQLite.
func (d TestDriver) IsServer() bool {
	return d.AdminDb != ""
}

// SkipConcurrentWrites skips the test on in-memory SQLite, whose shared cache
// fails concurrent writes with table locked errors.
func (d TestDriver) SkipConcurrentWrites(t *testing.T) {
	t.Helper()
	if d.Cfg.Driver == ldspec.SqliteDriver && d.Cfg.InMemorySQLite {
		t.Skip("the shared cache of in-memory sqlite fails concurrent writes with table locked errors")
	}
}

// RunTestsWithDbs runs the tests of the package and drops the test databases
// created on the servers; call it from TestMain.
func RunTestsWithDbs(m *testing.M) {
	code := m.Run()
	testDbs.dropAll()
	os.Exit(code)
}

// NewTestDBTemplate is the configuration of the driver for code creating
// its own databases from the url template; the SQLite files of the test are
// in a folder of its own. The databases created on a server must be dropped
// with DropTestDatabase.
func NewTestDBTemplate(t *testing.T, d TestDriver) ldspec.DBConfig {
	t.Helper()
	testCfg := d.Cfg
	if testCfg.Driver == ldspec.SqliteDriver && !testCfg.InMemorySQLite {
		testCfg.SQLiteDir = t.TempDir()
	}
	return testCfg.ToDBConfig()
}

// NewTestDBConfig is the configuration of a database of the test's own, in
// CoreDBName, so the tests, and the subtests within them, do not see each
// other's records. SQLite gets a new database each time; the servers reuse
// emptied databases from the pool with the name, as migrating a new one
// takes too long, mostly on cockroachdb. Use a pool for each schema, as the
// emptied databases keep their tables and schema version. Close the
// connections to the database before the test ends.
func NewTestDBConfig(t *testing.T, d TestDriver, pool string) ldspec.DBConfig {
	t.Helper()
	cfg := NewTestDBTemplate(t, d)
	if !d.IsServer() {
		cfg.CoreDBName = newTestDbName(pool)
		return cfg
	}

	key := d.Name + "/" + pool
	cfg.CoreDBName = testDbs.take(t, key, d, cfg, pool)
	// registered before the test opens the database, so it runs after the
	// test closes it
	t.Cleanup(func() {
		if clearTables(t, cfg) {
			testDbs.release(key, cfg.CoreDBName)
		}
	})
	return cfg
}

// DropTestDatabase drops the database with the name when the test ends, for
// the databases the code under test creates on a server; nothing is done on
// SQLite. Call it before the test connects, so it runs after the test closes
// its connections.
func DropTestDatabase(t *testing.T, d TestDriver, cfg ldspec.DBConfig, name string) {
	t.Helper()
	if !d.IsServer() {
		return
	}
	t.Cleanup(func() {
		admin, err := openDb(cfg.Driver, fmt.Sprintf(cfg.DbUrlTpl, d.AdminDb))
		if err != nil {
			t.Errorf("could not connect to drop the test database %s: %v", name, err)
			return
		}
		defer CloseDb(admin)
		if err := admin.Exec(fmt.Sprintf(d.DropSql, name)).Error; err != nil {
			t.Errorf("could not drop the test database %s: %v", name, err)
		}
	})
}

// TestDatabaseExists tells if the database with the name exists on the
// server of the driver; it returns false for SQLite.
func TestDatabaseExists(t *testing.T, d TestDriver, cfg ldspec.DBConfig, name string) bool {
	t.Helper()
	if !d.IsServer() {
		return false
	}
	admin, err := openDb(cfg.Driver, fmt.Sprintf(cfg.DbUrlTpl, d.AdminDb))
	require.NoError(t, err)
	defer CloseDb(admin)
	var exists bool
	require.NoError(t, admin.Raw("SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_database WHERE datname = ?)", name).Scan(&exists).Error)
	return exists
}

// UnreachableUrlTpl is a postgres url template of a server that is not
// running.
func UnreachableUrlTpl(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	return fmt.Sprintf("host=127.0.0.1 port=%d user=postgres password=x dbname=%%s sslmode=disable connect_timeout=1", port)
}

// ConflictOnCreate makes the next creates in the table fail, after the
// record is inserted, with the error the database sends when a transaction
// conflicts with another, so the transaction is rolled back and tried again
// as on a conflict. It returns the number of creates made in the table.
func ConflictOnCreate(t *testing.T, db *gorm.DB, table string, times int) *atomic.Int32 {
	t.Helper()
	name := "test:conflict_" + table
	var creates atomic.Int32
	var conflicts atomic.Int32
	require.NoError(t, db.Callback().Create().After("gorm:create").Register(name, func(tx *gorm.DB) {
		if tx.Error != nil || tx.Statement.Table != table {
			return
		}
		creates.Add(1)
		if conflicts.Add(1) <= int32(times) {
			_ = tx.AddError(&pgconn.PgError{Code: "40001", Message: "test conflict"})
		}
	}))
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove(name)
	})
	return &creates
}

// CloseDb closes the connections of the database.
func CloseDb(db *gorm.DB) {
	if sqlDb, err := db.DB(); err == nil {
		_ = sqlDb.Close()
	}
}

func newTestDbName(pool string) string {
	return strings.ToLower("ld" + pool + "_test_" + ldutils.GenerateRandomString(12))
}

// clearTables deletes the records of all the tables, so the database can be
// used by another test; it returns false when it could not. The schema
// version is kept, so the next test does not migrate the database again.
func clearTables(t *testing.T, cfg ldspec.DBConfig) bool {
	db, err := openDb(cfg.Driver, fmt.Sprintf(cfg.DbUrlTpl, cfg.CoreDBName))
	if err != nil {
		t.Logf("could not connect to clear the test database: %v", err)
		return false
	}
	defer CloseDb(db)
	tables, err := db.Migrator().GetTables()
	if err != nil {
		t.Logf("could not list the tables to clear: %v", err)
		return false
	}
	for i, table := range tables {
		if table == (ldspec.DbSchemaVersion{}).TableName() {
			tables = append(tables[:i], tables[i+1:]...)
			break
		}
	}
	clearForeignKeys(db)
	// a table referenced by a foreign key is cleared on a later pass, once
	// the tables referencing it are empty
	for pass := 0; pass < len(tables) && len(tables) > 0; pass++ {
		var failed []string
		for _, table := range tables {
			if err := db.Exec(fmt.Sprintf(`DELETE FROM "%s"`, table)).Error; err != nil {
				failed = append(failed, table)
			}
		}
		tables = failed
	}
	if len(tables) > 0 {
		t.Logf("could not clear the tables %v", tables)
		return false
	}
	return true
}

// clearForeignKeys sets the foreign keys of all the records to null, as the
// tables referencing each other, like the users and their calls, cannot be
// cleared one after the other. A column that cannot be null is left as is.
func clearForeignKeys(db *gorm.DB) {
	var columns []struct {
		TableName  string
		ColumnName string
	}
	err := db.Raw(`SELECT kcu.table_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema AND tc.table_name = kcu.table_name
		WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_schema = current_schema()`).Scan(&columns).Error
	if err != nil {
		return
	}
	for _, c := range columns {
		db.Exec(fmt.Sprintf(`UPDATE "%s" SET "%s" = NULL WHERE "%s" IS NOT NULL`, c.TableName, c.ColumnName, c.ColumnName))
	}
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
	cfg    ldspec.DBConfig
	names  []string
}

// take returns an empty database of the pool, creating it on the server of
// the driver when none is free.
func (p *testDbPool) take(t *testing.T, key string, d TestDriver, cfg ldspec.DBConfig, pool string) string {
	t.Helper()
	p.mu.Lock()
	if free := p.free[key]; len(free) > 0 {
		name := free[len(free)-1]
		p.free[key] = free[:len(free)-1]
		p.mu.Unlock()
		return name
	}
	p.mu.Unlock()

	name := newTestDbName(pool)
	admin, err := openDb(cfg.Driver, fmt.Sprintf(cfg.DbUrlTpl, d.AdminDb))
	require.NoError(t, err)
	defer CloseDb(admin)
	require.NoError(t, admin.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, name)).Error)

	p.mu.Lock()
	defer p.mu.Unlock()
	created, ok := p.created[key]
	if !ok {
		created = &poolDriver{driver: d, cfg: cfg}
		p.created[key] = created
	}
	created.names = append(created.names, name)
	return name
}

func (p *testDbPool) release(key string, name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.free[key] = append(p.free[key], name)
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
		CloseDb(admin)
	}
	p.created = map[string]*poolDriver{}
	p.free = map[string][]string{}
}

// openDb connects to the database without migrating it.
func openDb(driver ldspec.LDDatabaseDriver, url string) (*gorm.DB, error) {
	dialector, err := FindDialector(driver, url)
	if err != nil {
		return nil, err
	}
	return gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}
