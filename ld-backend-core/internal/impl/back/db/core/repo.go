package ldcore

import (
	"errors"
	"fmt"
	"ld-backend/internal/impl/back/db/conn"
	"ld-backend/internal/impl/back/db/core/query"
	"ld-backend/internal/spec"
	"log/slog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type coreRepoImpl struct {
	coreDb    *gorm.DB
	coreQuery *query.Query
}

func NewCoreRepo(cfg ldspec.DBConfig) (ldspec.CoreRepo, error) {
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

// Db is the connection to the core database.
func (t *coreRepoImpl) Db() *gorm.DB {
	return t.coreDb
}

func (t *coreRepoImpl) FindTenant(id string) (*ldspec.DbTenant, error) {
	return t.coreQuery.DbTenant.
		Where(t.coreQuery.DbTenant.BusinessId.Eq(id)).
		First()
}

func (t *coreRepoImpl) SaveTenant(result *ldspec.DbTenant) error {
	return ldconn.Transaction(t.coreDb, func(tx *gorm.DB) error {
		return tx.Save(result).Error
	})
}

// FindCompanies returns all the GHL companies with the app installed.
func (t *coreRepoImpl) FindCompanies() ([]*ldspec.DbGHLCompany, error) {
	return t.coreQuery.DbGHLCompany.Find()
}

func (t *coreRepoImpl) FindAllLocations() ([]*ldspec.DbGHLLocation, error) {
	return t.coreQuery.DbGHLLocation.Find()
}

// FindCompany returns the companies with the GHL company id; company ids are
// unique, so it has one item at most.
func (t *coreRepoImpl) FindCompany(companyId string) (*ldspec.DbGHLCompany, error) {
	c := t.coreQuery.DbGHLCompany
	return c.Where(c.CompanyId.Eq(companyId)).First()
}

func (t *coreRepoImpl) FindByAuthCode(authCode string) (*ldspec.DbGHLCompany, *ldspec.DbGHLLocation, error) {
	ql := t.coreQuery
	company, err := ql.DbGHLCompany.Where(ql.DbGHLCompany.InstallAuthCode.Eq(authCode)).First()
	if err == nil {
		return company, nil, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	location, err := ql.DbGHLLocation.Where(ql.DbGHLLocation.InstallAuthCode.Eq(authCode)).First()
	if err != nil {
		return nil, nil, err
	}
	return nil, location, nil
}

// SaveCompanyAppInstall saves the company installed with the authorization
// code. An authorization code is used once, so when a company was already
// saved with it, that company is returned unchanged; otherwise the company
// is saved with the code, updating the one with the same company id.
func (t *coreRepoImpl) SaveCompanyAppInstall(authCode string, company *ldspec.DbGHLCompany) (*ldspec.DbGHLCompany, error) {
	var result *ldspec.DbGHLCompany
	err := ldconn.Transaction(t.coreDb, func(tx *gorm.DB) error {
		c := query.Use(tx).DbGHLCompany
		existing, err := c.Where(c.InstallAuthCode.Eq(authCode)).First()
		if err == nil {
			result = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		company.InstallAuthCode = &authCode
		if err := saveCompany(tx, company); err != nil {
			return err
		}
		result = company
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// SaveCompany saves the company; a new record updates the existing one with
// the same company id.
func (t *coreRepoImpl) SaveCompany(company *ldspec.DbGHLCompany) error {
	return ldconn.Transaction(t.coreDb, func(tx *gorm.DB) error {
		return saveCompany(tx, company)
	})
}

func saveCompany(tx *gorm.DB, company *ldspec.DbGHLCompany) error {
	if company.ID != 0 {
		return tx.Save(company).Error
	}
	// a new record updates the company with the same id, restoring it when
	// deleted, and gets its id and creation time back
	c := query.Use(tx).DbGHLCompany
	updates := clause.AssignmentColumns([]string{
		string(c.Name.ColumnName()),
		string(c.InstallAuthCode.ColumnName()),
		string(c.AccessToken.ColumnName()),
		string(c.RefreshToken.ColumnName()),
		string(c.UserId.ColumnName()),
		string(c.ExpiresIn.ColumnName()),
		string(c.UpdatedAt.ColumnName()),
	})
	updates = append(updates, clause.Assignment{Column: clause.Column{Name: string(c.DeletedAt.ColumnName())}, Value: nil})
	return c.Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: string(c.CompanyId.ColumnName())}},
			DoUpdates: updates,
		},
		clause.Returning{},
	).Create(company)
}

// FindLocations returns the locations of the company.
func (t *coreRepoImpl) FindLocations(companyId string) ([]*ldspec.DbGHLLocation, error) {
	l := t.coreQuery.DbGHLLocation
	return l.Where(l.CompanyId.Eq(companyId)).Find()
}

// FindLocation finds the location by its GHL location id.
func (t *coreRepoImpl) FindLocation(locationId string) (*ldspec.DbGHLLocation, error) {
	l := t.coreQuery.DbGHLLocation
	return l.Where(l.LocationId.Eq(locationId)).First()
}

// SaveLocation saves the location; a new record updates the existing one
// with the same location id.
func (t *coreRepoImpl) SaveLocation(location *ldspec.DbGHLLocation) error {
	return ldconn.Transaction(t.coreDb, func(tx *gorm.DB) error {
		return saveLocation(tx, location)
	})
}

// SaveLocationAppInstall saves the location installed with the authorization
// code. An authorization code is used once, so when a location was already
// saved with it, that location is returned unchanged; otherwise the location
// is saved with the code, updating the one with the same location id.
func (t *coreRepoImpl) SaveLocationAppInstall(authCode string, location *ldspec.DbGHLLocation) (*ldspec.DbGHLLocation, error) {
	var result *ldspec.DbGHLLocation
	err := ldconn.Transaction(t.coreDb, func(tx *gorm.DB) error {
		l := query.Use(tx).DbGHLLocation
		existing, err := l.Where(l.InstallAuthCode.Eq(authCode)).First()
		if err == nil {
			result = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		location.InstallAuthCode = &authCode
		if err := saveLocation(tx, location); err != nil {
			return err
		}
		result = location
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func saveLocation(tx *gorm.DB, location *ldspec.DbGHLLocation) error {
	if location.ID != 0 {
		return tx.Save(location).Error
	}
	// a new record updates the location with the same id, restoring it when
	// deleted, and gets its id and creation time back
	l := query.Use(tx).DbGHLLocation
	updates := clause.AssignmentColumns([]string{
		string(l.CompanyId.ColumnName()),
		string(l.Name.ColumnName()),
		string(l.AccessToken.ColumnName()),
		string(l.RefreshToken.ColumnName()),
		string(l.InstallType.ColumnName()),
		string(l.UserId.ColumnName()),
		string(l.ExpiresIn.ColumnName()),
		string(l.InstallAuthCode.ColumnName()),
		string(l.UpdatedAt.ColumnName()),
	})
	updates = append(updates, clause.Assignment{Column: clause.Column{Name: string(l.DeletedAt.ColumnName())}, Value: nil})
	return l.Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: string(l.LocationId.ColumnName())}},
			DoUpdates: updates,
		},
		clause.Returning{},
	).Create(location)
}

func (t *coreRepoImpl) FindServiceAccount(clientId *string) (*ldspec.DbServiceAccount, error) {
	if clientId == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return t.coreQuery.DbServiceAccount.
		Where(t.coreQuery.DbServiceAccount.ClientId.Eq(*clientId)).
		First()
}

func (t *coreRepoImpl) SaveServiceAccount(account *ldspec.DbServiceAccount) error {
	return ldconn.Transaction(t.coreDb, func(tx *gorm.DB) error {
		return tx.Save(account).Error
	})
}

func (t *coreRepoImpl) FindNode(nodeId string) (*ldspec.DbServerNode, error) {
	return t.coreQuery.DbServerNode.
		Where(t.coreQuery.DbServerNode.NodeId.Eq(nodeId)).
		First()
}

func (t *coreRepoImpl) FindAllNodes() ([]*ldspec.DbServerNode, error) {
	return t.coreQuery.DbServerNode.Find()
}

func (t *coreRepoImpl) SaveNode(node *ldspec.DbServerNode) error {
	return ldconn.Transaction(t.coreDb, func(tx *gorm.DB) error {
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

// FindPayments returns the payments of the location, the newest first.
func (t *coreRepoImpl) FindPayments(locationId string) ([]*ldspec.DbPaymentRecord, error) {
	p := t.coreQuery.DbPaymentRecord
	return p.
		Where(p.LocationId.Eq(locationId)).
		Order(p.DateTime.Desc(), p.ID.Desc()).
		Find()
}

// SavePayment creates or updates the payment record.
func (t *coreRepoImpl) SavePayment(payment *ldspec.DbPaymentRecord) error {
	return ldconn.Transaction(t.coreDb, func(tx *gorm.DB) error {
		return tx.Save(payment).Error
	})
}

// FindPaidSubscription returns the paid subscriptions of the location.
func (t *coreRepoImpl) FindPaidSubscription(locationId string) (*ldspec.DbPaidSubscription, error) {
	s := t.coreQuery.DbPaidSubscription
	return s.
		Where(s.LocationId.Eq(locationId)).
		Order(s.ID).
		First()
}

// SavePaidSubscription saves the paid subscription; a new record updates the
// existing one with the same location id.
func (t *coreRepoImpl) SavePaidSubscription(paidSubscription *ldspec.DbPaidSubscription) error {
	return ldconn.Transaction(t.coreDb, func(tx *gorm.DB) error {
		if paidSubscription.ID != 0 {
			return tx.Save(paidSubscription).Error
		}
		// a new record updates the subscription of the same location,
		// restoring it when deleted, and gets its id and creation time back
		s := query.Use(tx).DbPaidSubscription
		updates := clause.AssignmentColumns([]string{
			string(s.BusinessId.ColumnName()),
			string(s.StripeCustomerId.ColumnName()),
			string(s.Status.ColumnName()),
			string(s.MonthlyAmount.ColumnName()),
			string(s.PerUserAmount.ColumnName()),
			string(s.AmountDue.ColumnName()),
			string(s.Accounts.ColumnName()),
			string(s.LastPayment.ColumnName()),
			string(s.UpdatedAt.ColumnName()),
		})
		updates = append(updates, clause.Assignment{Column: clause.Column{Name: string(s.DeletedAt.ColumnName())}, Value: nil})
		return s.Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: string(s.LocationId.ColumnName())}},
				DoUpdates: updates,
			},
			clause.Returning{},
		).Create(paidSubscription)
	})
}

func connect(driver ldspec.LDDatabaseDriver, url string) (*gorm.DB, error) {
	slog.Info("connecting to core database", "module", "db/core")
	dialector, err := ldconn.FindDialector(driver, url)
	if err != nil {
		return nil, fmt.Errorf("could not find dialector: %w", err)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	err = ldconn.MigrateIfNeeded(db, "db/core", func() error {
		return db.AutoMigrate(ldspec.DbCoreTables...)
	})
	if err != nil {
		return nil, fmt.Errorf("could not migrate database: %w", err)
	}

	return db, nil
}
