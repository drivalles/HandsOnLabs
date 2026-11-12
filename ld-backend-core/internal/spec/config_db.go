package ldspec

import "fmt"

// DBConfig is the configuration to connect to the core and tenant databases.
type DBConfig struct {
	// Driver is the database engine used for the core and tenant databases.
	Driver LDDatabaseDriver
	// CoreDBName is the name of the core database, put into DbUrlTpl.
	CoreDBName string
	// DbUrlTpl is the connection url with a %s placeholder for the database
	// name; it is used for the core database and for each tenant database.
	DbUrlTpl string
}

// LDDatabaseDriver is the database engine used by the repositories.
type LDDatabaseDriver string

const (
	// PostgresDriver connects to PostgreSQL; tenant databases are created on
	// the server when missing.
	PostgresDriver LDDatabaseDriver = "postgres"
	// SqliteDriver connects to SQLite files, used for tests and local runs.
	SqliteDriver LDDatabaseDriver = "sqlite"
	// CockroachDBDriver connects to CockroachDB with the postgres driver.
	CockroachDBDriver LDDatabaseDriver = "cockroachdb"
)

// DBTestConfig are the databases of the tests.
type DBTestConfig struct {
	Driver LDDatabaseDriver
	// InMemorySQLite keeps the SQLite databases in memory, without files.
	InMemorySQLite bool
	// SQLiteDir is the folder of the SQLite database files.
	SQLiteDir      string
	CoreDbName     string
	TestTenantName string
	// PostgresTestIp is the address of the server, or the addresses of the
	// nodes of a cluster separated by commas.
	TestServerIp       string
	TestServerPort     int
	TestServerUser     string
	TestServerPassword string
}

func (tCfg DBTestConfig) ToDBConfig() DBConfig {
	cfg := DBConfig{
		Driver:     tCfg.Driver,
		CoreDBName: tCfg.CoreDbName,
	}
	if tCfg.Driver == SqliteDriver {
		if tCfg.InMemorySQLite {
			cfg.DbUrlTpl = "file:%s?mode=memory&cache=shared&loc=auto"
		} else {
			cfg.DbUrlTpl = fmt.Sprintf("%s/%s.db?_journal_mode=WAL&_busy_timeout=5000&_txlock=immediate", tCfg.SQLiteDir, "%s")
		}
	} else if tCfg.Driver == PostgresDriver {
		cfg.DbUrlTpl = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			tCfg.TestServerIp, tCfg.TestServerPort, tCfg.TestServerUser, tCfg.TestServerPassword, "%s")
	} else if tCfg.Driver == CockroachDBDriver {
		// the test server runs in secure mode, so it only takes TLS
		cfg.DbUrlTpl = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
			tCfg.TestServerIp, tCfg.TestServerPort, tCfg.TestServerUser, tCfg.TestServerPassword, "%s")
	} else {
		panic(fmt.Sprintf("unknown driver %s", tCfg.Driver))
	}
	return cfg
}
