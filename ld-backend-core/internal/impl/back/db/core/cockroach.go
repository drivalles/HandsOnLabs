package core

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

// cockroachDialector connects to CockroachDB with the postgres driver, which
// it is wire compatible with, fixing the migrations where they differ.
type cockroachDialector struct {
	postgres.Dialector
}

func newCockroachDialector(url string) gorm.Dialector {
	return cockroachDialector{postgres.Dialector{Config: &postgres.Config{DSN: cockroachUrl(url)}}}
}

func (d cockroachDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return cockroachMigrator{postgres.Migrator{Migrator: migrator.Migrator{Config: migrator.Config{
		DB:                          db,
		Dialector:                   d,
		CreateIndexAfterCreateTable: true,
	}}}}
}

type cockroachMigrator struct {
	postgres.Migrator
}

// MigrateColumnUnique keeps the unique indexes. CockroachDB reports the
// columns of a uniqueIndex as unique, so postgres would try to drop a unique
// constraint that was never created each time the tables are migrated.
func (m cockroachMigrator) MigrateColumnUnique(value interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	if unique, ok := columnType.Unique(); ok && unique && !field.Unique {
		return nil
	}
	return m.Migrator.MigrateColumnUnique(value, field, columnType)
}

// GetTypeAliases knows char columns by their bpchar type name; without it
// each migration alters their type again, which CockroachDB refuses for the
// columns of an index.
func (m cockroachMigrator) GetTypeAliases(databaseTypeName string) []string {
	if databaseTypeName == "bpchar" {
		return []string{"char", "character"}
	}
	return m.Migrator.GetTypeAliases(databaseTypeName)
}

// cockroachUrl makes the serial ids sequential: by default CockroachDB fills
// them with unique_rowid(), whose values are too big for javascript numbers.
func cockroachUrl(url string) string {
	const param = "serial_normalization"
	if strings.Contains(url, param) {
		return url
	}
	if strings.Contains(url, "://") {
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		return url + sep + param + "=sql_sequence"
	}
	return url + " " + param + "=sql_sequence"
}

// txRetries is how many times a transaction is tried before giving up.
const txRetries = 5

// transaction runs fn in a transaction, running it again when the database
// asks to retry it. CockroachDB runs the transactions serializable and fails
// the ones conflicting with others with a retry error, so fn must be safe to
// run more than once.
func transaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	var err error
	for attempt := 1; attempt <= txRetries; attempt++ {
		err = db.Transaction(fn)
		if !isRetryable(err) {
			return err
		}
		time.Sleep(time.Duration(attempt*10+rand.Intn(10)) * time.Millisecond)
	}
	return err
}

// isRetryable tells if the error is a serialization failure, sent when a
// transaction conflicts with another and can be tried again.
func isRetryable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40001"
}
