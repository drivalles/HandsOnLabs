// Package dbconn connects the repositories to the database engines.
package ldconn

import (
	"fmt"
	"ld-backend/internal/spec"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// FindDialector returns the gorm dialector of the driver for the url.
func FindDialector(driver ldspec.LDDatabaseDriver, url string) (gorm.Dialector, error) {
	switch driver {
	case ldspec.PostgresDriver:
		return postgres.Open(url), nil
	case ldspec.CockroachDBDriver:
		return newCockroachDialector(url), nil
	case ldspec.SqliteDriver:
		return sqlite.Open(url), nil
	}
	return nil, fmt.Errorf("unsupported driver %s for url %s", driver, url)
}
