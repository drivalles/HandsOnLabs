package ldtenant

import (
	"errors"
	"fmt"
	"ld-backend/internal/impl/back/db/conn"
	"ld-backend/internal/spec"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type tenantDbManager struct {
	coreRepo ldspec.CoreRepo
	cfg      ldspec.DBConfig
}

func newTenantDbManager(coreRepo ldspec.CoreRepo, cfg ldspec.DBConfig) *tenantDbManager {
	return &tenantDbManager{coreRepo: coreRepo, cfg: cfg}
}

func (mang *tenantDbManager) find(businessId string) (*gorm.DB, error) {
	slog.Info(fmt.Sprintf("finding or creating tenant for business %s", businessId), "module", "db/tenant")
	tenant, err := mang.coreRepo.FindTenant(businessId)
	if err != nil {
		return nil, err
	}
	return connectUsingTenant(tenant)
}

func (mang *tenantDbManager) findOrCreate(businessId string) (*gorm.DB, error) {
	slog.Info(fmt.Sprintf("finding or creating tenant for business %s", businessId), "module", "db/tenant")
	tenant, err := mang.coreRepo.FindTenant(businessId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Warn(fmt.Sprintf("tenant for business %s returned error %v", businessId, err), "module", "db/tenant")
			return nil, err
		}
		tenant, err = mang.createNewTenant(businessId)
	}
	if err != nil {
		return nil, fmt.Errorf("could not find or create tenant: %w", err)
	}
	return connectUsingTenant(tenant)
}

func (mang *tenantDbManager) createNewTenant(businessId string) (*ldspec.DbTenant, error) {
	slog.Info(fmt.Sprintf("creating new tenant for business %s", businessId), "module", "db/tenant")
	url := mang.cfg.DbUrlTpl
	dbName := tenantDbName(businessId)
	if strings.Contains(mang.cfg.DbUrlTpl, "%s") {
		url = fmt.Sprintf(mang.cfg.DbUrlTpl, dbName)
	}
	result := &ldspec.DbTenant{
		BusinessId: businessId,
		Status:     ldspec.LDActive,
		Driver:     string(mang.cfg.Driver),
		Url:        url,
		DbName:     dbName,
		Title:      businessId,
	}
	err := mang.createTenantDatabase(result.DbName)
	if err != nil {
		return nil, err
	}
	err = mang.coreRepo.SaveTenant(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// tenantDbName is the name of the tenant database of the business.
func tenantDbName(businessId string) string {
	return "loc_" + strings.ToLower(businessId)
}

func (mang *tenantDbManager) createTenantDatabase(dbName string) error {
	slog.Info(fmt.Sprintf("creating tenant database %s", dbName), "module", "db/tenant")
	if mang.cfg.Driver == ldspec.PostgresDriver || mang.cfg.Driver == ldspec.CockroachDBDriver {
		return createServerDatabase(mang.cfg.Driver, mang.cfg.DbUrlTpl, dbName)
	} else if mang.cfg.Driver == ldspec.SqliteDriver {
		return nil
	}
	return fmt.Errorf("unsupported driver %s for tenant %s", mang.cfg.Driver, dbName)
}

func connectUsingTenant(tenant *ldspec.DbTenant) (*gorm.DB, error) {
	if tenant.Status != ldspec.LDActive {
		return nil, fmt.Errorf("tenant %s is not active", tenant.BusinessId)
	}
	slog.Info(fmt.Sprintf("connecting to tenant database %s", tenant.DbName), "module", "db/tenant")
	driver := ldspec.LDDatabaseDriver(tenant.Driver)
	dialector, err := ldconn.FindDialector(driver, tenant.Url)
	if err != nil {
		return nil, err
	}
	db, err := connect(dialector)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func connect(dialector gorm.Dialector) (*gorm.DB, error) {
	slog.Info("connecting to tenant database", "module", "db/tenant")
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}
	err = ldconn.MigrateIfNeeded(db, "db/tenant", func() error {
		return migrate(dialector, db)
	})
	if err != nil {
		closeDb(db)
		return nil, err
	}

	slog.Info("tenant database connected", "module", "db/tenant")
	return db, nil
}

// migrate creates the tables without their foreign keys first, as they are
// migrated one at a time and may reference tables not created yet, then
// adds the foreign keys.
func migrate(dialector gorm.Dialector, db *gorm.DB) error {
	migDb, err := gorm.Open(dialector, &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}
	defer closeDb(migDb)
	if err = doMigration(migDb); err != nil {
		return err
	}
	return doMigration(db)
}

func closeDb(db *gorm.DB) {
	if sqlDb, err := db.DB(); err == nil {
		_ = sqlDb.Close()
	}
}

func doMigration(db *gorm.DB) error {
	slog.Info("migrating tenant database", "module", "db/tenant")
	for _, m := range ldspec.DbTenantTables {
		err := db.AutoMigrate(&m)
		if err != nil {
			return fmt.Errorf("could not migrate database: %w", err)
		}
	}
	return nil
}

// createServerDatabase creates the tenant database on a postgres or
// cockroachdb server, connecting to their postgres database to do it.
func createServerDatabase(driver ldspec.LDDatabaseDriver, urlTemplate string, dbName string) error {
	dialector, err := ldconn.FindDialector(driver, fmt.Sprintf(urlTemplate, "postgres"))
	if err != nil {
		return err
	}
	slog.Info("connecting to the server to create tenant database", "module", "db/tenant")
	db, err := gorm.Open(dialector)
	if err != nil {
		return err
	}
	exists := false
	tx := db.Raw(fmt.Sprintf("SELECT EXISTS ( SELECT 1 FROM pg_catalog.pg_database WHERE datname = '%s');", dbName))
	if tx.Error != nil {
		return tx.Error
	}
	tx.Scan(&exists)
	if !exists {
		slog.Info("creating tenant database", "module", "db/tenant")
		tx = db.Exec("CREATE DATABASE " + strings.ToLower(dbName))
		if tx.Error != nil {
			return tx.Error
		}
		err = pingTenantDatabase(driver, urlTemplate, dbName)
		if err != nil {
			return err
		}
		slog.Info("created database for tenant " + dbName)
	}
	return nil
}

func pingTenantDatabase(driver ldspec.LDDatabaseDriver, urlTemplate string, dbName string) error {
	slog.Info("pinging tenant database", "module", "db/tenant")
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	count := 0
	for ; true; <-ticker.C {
		slog.Info(fmt.Sprintf("pinging tenant database count %v", count), "module", "db/tenant")
		if count > 5 {
			break
		}
		count++
		err := checkTenantDbConnection(driver, urlTemplate, dbName)
		if err != nil {
			slog.Warn(fmt.Sprintf("tenant database not ready yet, retrying... %v", err), "module", "db/tenant")
			continue
		}

		return nil
	}
	return errors.New("timeout pinging tenant database")
}

func checkTenantDbConnection(driver ldspec.LDDatabaseDriver, urlTemplate string, dbName string) error {
	slog.Info("connecting to the server to test tenant database", "module", "db/tenant")
	dialector, err := ldconn.FindDialector(driver, fmt.Sprintf(urlTemplate, dbName))
	if err != nil {
		return err
	}
	db, err := gorm.Open(dialector)
	if err != nil {
		return fmt.Errorf("could not connect to db: %w", err)
	}
	sqlDb, err := db.DB()
	if err != nil {
		return fmt.Errorf("could not get sql db: %w", err)
	}
	defer func() {
		err = sqlDb.Close()
		if err != nil {
			slog.Warn("closing database connection", "module", "db/tenant")
		}
	}()
	err = sqlDb.Ping()
	if err != nil {
		return fmt.Errorf("could not ping tenant database: %w", err)
	}
	return nil
}
