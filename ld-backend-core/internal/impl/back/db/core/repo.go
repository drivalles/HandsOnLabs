package core

import (
	"fmt"
	"ld-backend/internal/impl/back/db/core/query"
	"ld-backend/internal/spec"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type coreRepoImpl struct {
	coreDb    *gorm.DB
	coreQuery *query.Query
}

func NewCoreRepo(cfg spec.DBConfig) (spec.CoreRepo, error) {
	driver := cfg.Driver
	url := fmt.Sprintf(cfg.DbUrlTpl, cfg.CoreDBName)
	db, err := connect(driver, url)
	if err != nil {
		return nil, err
	}
	coreQuery := query.Use(db)
	result := coreRepoImpl{
		coreDb:    db,
		coreQuery: coreQuery,
	}
	return &result, nil
}

func (t *coreRepoImpl) FindTenant(id string) (*spec.DbTenant, error) {
	return t.coreQuery.DbTenant.
		Where(t.coreQuery.DbTenant.BusinessId.Eq(id)).
		First()
}

func (t *coreRepoImpl) SaveTenant(result *spec.DbTenant) error {
	return transaction(t.coreDb, func(tx *gorm.DB) error {
		return tx.Save(result).Error
	})
}

func (t *coreRepoImpl) FindCompanies() ([]string, error) {
	result, err := t.coreQuery.DbGHLAppInstallation.
		Distinct(t.coreQuery.DbGHLAppInstallation.CompanyId).
		Find()
	if err != nil {
		return nil, err
	}
	companyIds := make([]string, 0, len(result))
	for _, c := range result {
		if c.CompanyId == nil {
			continue
		}
		companyIds = append(companyIds, *c.CompanyId)
	}
	return companyIds, nil
}

func (t *coreRepoImpl) GetLatestInstall(companyId string) (*spec.DbGHLAppInstallation, error) {
	item, err := t.coreQuery.DbGHLAppInstallation.
		Where(t.coreQuery.DbGHLAppInstallation.CompanyId.Eq(companyId)).
		Where(t.coreQuery.DbGHLAppInstallation.LocationId.IsNull()).
		Order(t.coreQuery.DbGHLAppInstallation.CreatedAt.Desc()).
		First()
	return item, err
}

func (t *coreRepoImpl) GetLatestAccess(locationId string) (*spec.DbGHLAppInstallation, error) {
	items, err := t.coreQuery.DbGHLAppInstallation.
		Where(t.coreQuery.DbGHLAppInstallation.LocationId.Eq(locationId)).
		Order(t.coreQuery.DbGHLAppInstallation.CreatedAt.Desc()).
		First()
	return items, err
}

func (t *coreRepoImpl) FindByCompanyAndAuthCode(companyId string, authCode string) (*spec.DbGHLAppInstallation, error) {
	return t.coreQuery.DbGHLAppInstallation.
		Where(t.coreQuery.DbGHLAppInstallation.CompanyId.Eq(companyId)).
		Where(t.coreQuery.DbGHLAppInstallation.AuthorizationCode.Eq(authCode)).
		Where(t.coreQuery.DbGHLAppInstallation.LocationId.IsNull()).
		Order(t.coreQuery.DbGHLAppInstallation.CreatedAt.Desc()).
		First()
}

func (t *coreRepoImpl) FindByAuthCode(authCode string) (*spec.DbGHLAppInstallation, error) {
	return t.coreQuery.DbGHLAppInstallation.
		Where(t.coreQuery.DbGHLAppInstallation.AuthorizationCode.Eq(authCode)).
		Where(t.coreQuery.DbGHLAppInstallation.LocationId.IsNull()).
		Order(t.coreQuery.DbGHLAppInstallation.CreatedAt.Desc()).
		First()
}

func (t *coreRepoImpl) SaveAppInstall(install *spec.DbGHLAppInstallation) error {
	return transaction(t.coreDb, func(tx *gorm.DB) error {
		return tx.Save(install).Error
	})
}

func (t *coreRepoImpl) FindAllAppInstalls() ([]*spec.DbGHLAppInstallation, error) {
	return t.coreQuery.DbGHLAppInstallation.Find()
}

func (t *coreRepoImpl) CleanAppInstalls(date time.Time) error {
	return transaction(t.coreDb, func(tx *gorm.DB) error {
		q := query.Use(tx)
		_, err := q.DbGHLAppInstallation.Unscoped().
			Where(q.DbGHLAppInstallation.CreatedAt.Lt(date)).
			Delete()
		return err
	})
}

func (t *coreRepoImpl) FindInstalledLocations(companyId string) ([]*spec.DbGHLInstalledLocation, error) {
	install, err := t.GetLatestInstall(companyId)
	if err != nil {
		return nil, err
	}
	return t.coreQuery.DbGHLInstalledLocation.
		Where(t.coreQuery.DbGHLInstalledLocation.CompanyId.Eq(*install.CompanyId)).
		Find()
}

func (t *coreRepoImpl) FindInstalledLocation(locationId string) (*spec.DbGHLInstalledLocation, error) {
	return t.coreQuery.DbGHLInstalledLocation.
		Where(t.coreQuery.DbGHLInstalledLocation.LocationId.Eq(locationId)).
		First()
}

func (t *coreRepoImpl) SaveInstalledLocation(install *spec.DbGHLInstalledLocation) error {
	return transaction(t.coreDb, func(tx *gorm.DB) error {
		if install.ID != 0 {
			return tx.Save(install).Error
		}
		// a new record updates the location with the same id, restoring it
		// when deleted, and gets its id and creation time back
		l := query.Use(tx).DbGHLInstalledLocation
		updates := clause.AssignmentColumns([]string{
			string(l.CompanyId.ColumnName()),
			string(l.IsInstalled.ColumnName()),
			string(l.Name.ColumnName()),
			string(l.VersionId.ColumnName()),
			string(l.Address.ColumnName()),
			string(l.InstalledAt.ColumnName()),
			string(l.UpdatedAt.ColumnName()),
		})
		updates = append(updates, clause.Assignment{Column: clause.Column{Name: string(l.DeletedAt.ColumnName())}, Value: nil})
		return l.Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: string(l.LocationId.ColumnName())}},
				DoUpdates: updates,
			},
			clause.Returning{},
		).Create(install)
	})
}

func (t *coreRepoImpl) FindServiceAccount(clientId *string) (*spec.DbServiceAccount, error) {
	if clientId == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return t.coreQuery.DbServiceAccount.
		Where(t.coreQuery.DbServiceAccount.ClientId.Eq(*clientId)).
		First()
}

func (t *coreRepoImpl) SaveServiceAccount(account *spec.DbServiceAccount) error {
	return transaction(t.coreDb, func(tx *gorm.DB) error {
		return tx.Save(account).Error
	})
}

func (t *coreRepoImpl) FindNode(nodeId string) (*spec.DbServerNode, error) {
	return t.coreQuery.DbServerNode.
		Where(t.coreQuery.DbServerNode.NodeId.Eq(nodeId)).
		First()
}

func (t *coreRepoImpl) FindAllNodes() ([]*spec.DbServerNode, error) {
	return t.coreQuery.DbServerNode.Find()
}

func (t *coreRepoImpl) SaveNode(node *spec.DbServerNode) error {
	return transaction(t.coreDb, func(tx *gorm.DB) error {
		if node.ID != 0 {
			return tx.Save(node).Error
		}
		// a new record updates the node with the same id, restoring it when
		// deleted, and gets its id and creation time back
		n := query.Use(tx).DbServerNode
		updates := clause.AssignmentColumns([]string{
			string(n.Status.ColumnName()),
			string(n.LastHeartbeat.ColumnName()),
			string(n.UpdatedAt.ColumnName()),
		})
		updates = append(updates, clause.Assignment{Column: clause.Column{Name: string(n.DeletedAt.ColumnName())}, Value: nil})
		return n.Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: string(n.NodeId.ColumnName())}},
				DoUpdates: updates,
			},
			clause.Returning{},
		).Create(node)
	})
}

func connect(driver spec.LDDatabaseDriver, url string) (*gorm.DB, error) {
	slog.Info("connecting to core database", "module", "db/core")
	dialector, err := findDialector(driver, url)
	if err != nil {
		return nil, fmt.Errorf("could not find dialector: %w", err)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	err = db.AutoMigrate(spec.DbCoreTables...)
	if err != nil {
		return nil, fmt.Errorf("could not migrate database: %w", err)
	}

	return db, nil
}

func findDialector(driver spec.LDDatabaseDriver, url string) (gorm.Dialector, error) {
	var dialector gorm.Dialector
	if driver == spec.PostgresDriver {
		dialector = postgres.Open(url)
	} else if driver == spec.CockroachDBDriver {
		dialector = newCockroachDialector(url)
	} else if driver == spec.SqliteDriver {
		dialector = sqlite.Open(url)
	} else {
		return nil, fmt.Errorf("unsupported driver %s for url %s", driver, url)
	}
	return dialector, nil
}
