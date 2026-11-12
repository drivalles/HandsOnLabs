package ldconn

import (
	"errors"
	"fmt"
	ldspec "ld-backend/internal/spec"
	"log/slog"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// MigrateIfNeeded runs migrate when the schema of the database is older than
// ldspec.DB_SCHEMA_VERSION, recording the new version after it. Checking the
// version is much faster than migrating, mostly on CockroachDB, where
// checking the tables takes seconds.
func MigrateIfNeeded(db *gorm.DB, database string, migrate func() error) error {
	current, err := SchemaVersion(db)
	if err != nil {
		return err
	}
	if current != "" {
		diff, err := CompareVersions(current, ldspec.DB_SCHEMA_VERSION)
		if err != nil {
			return err
		}
		if diff == 0 {
			slog.Info(fmt.Sprintf("schema version %s is current, skipping migration, for database: %s", current, database), "module", "db/conn")
			return nil
		}
		if diff > 0 {
			// a newer server migrated it, as while it is deployed
			slog.Warn(fmt.Sprintf("schema version %s is newer than %s, skipping migration, for database: %s", current, ldspec.DB_SCHEMA_VERSION, database), "module", "db/conn")
			return nil
		}
	}

	slog.Info(fmt.Sprintf("migrating schema from version %q to %s, for database: %s", current, ldspec.DB_SCHEMA_VERSION, database), "module", "db/conn")
	if err := migrate(); err != nil {
		return err
	}
	if err := db.AutoMigrate(&ldspec.DbSchemaVersion{}); err != nil {
		return fmt.Errorf("could not migrate the schema versions: %w", err)
	}
	if err := db.Create(&ldspec.DbSchemaVersion{Version: ldspec.DB_SCHEMA_VERSION}).Error; err != nil {
		return fmt.Errorf("could not save the schema version: %w", err)
	}
	return nil
}

// SchemaVersion returns the version of the schema of the database, empty
// when it was never migrated with a version.
func SchemaVersion(db *gorm.DB) (string, error) {
	if !db.Migrator().HasTable(&ldspec.DbSchemaVersion{}) {
		return "", nil
	}
	var latest ldspec.DbSchemaVersion
	err := db.Order("id desc").First(&latest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("could not read the schema version: %w", err)
	}
	return latest.Version, nil
}

// compareVersions compares two dotted versions number by number, so 1.10.0
// is newer than 1.9.0; it returns -1, 0 or 1.
func CompareVersions(a string, b string) (int, error) {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		aNum, err := versionPart(aParts, i)
		if err != nil {
			return 0, fmt.Errorf("invalid schema version %q: %w", a, err)
		}
		bNum, err := versionPart(bParts, i)
		if err != nil {
			return 0, fmt.Errorf("invalid schema version %q: %w", b, err)
		}
		if aNum != bNum {
			if aNum < bNum {
				return -1, nil
			}
			return 1, nil
		}
	}
	return 0, nil
}

func versionPart(parts []string, i int) (int, error) {
	if i >= len(parts) {
		return 0, nil
	}
	return strconv.Atoi(parts[i])
}
