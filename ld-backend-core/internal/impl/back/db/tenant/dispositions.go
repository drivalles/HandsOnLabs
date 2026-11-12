package ldtenant

import (
	"errors"
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"

	"gorm.io/gorm"
)

func (c *tenantsRepoImpl) FindDispositions() ([]*ldspec.DbDisposition, error) {
	return c.tenantDb.DbDisposition.
		Order(c.tenantDb.DbDisposition.Name).
		Find()
}

func (c *tenantsRepoImpl) FindDispositionById(id uint) (*ldspec.DbDisposition, error) {
	return c.tenantDb.DbDisposition.
		Where(c.tenantDb.DbDisposition.ID.Eq(id)).
		First()
}

func (c *tenantsRepoImpl) SaveDisposition(disposition *ldspec.DbDisposition) error {
	// isNew is decided before the transaction: Create sets the ID even when
	// the commit fails, and a retry must look up and create it again.
	isNew := disposition.ID == 0
	return c.transaction(func(q *tenantQuery.Query) error {
		tq := q.DbDisposition
		if isNew {
			disposition.ID = 0
		}
		id := disposition.ID
		if id == 0 && disposition.GHLId != nil {
			oldDisposition, err := tq.Where(tq.GHLId.Eq(*disposition.GHLId)).First()
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if oldDisposition != nil {
				id = oldDisposition.ID
			}
		}

		if id == 0 {
			return tq.Create(disposition)
		}

		info, err := tq.
			Select(tq.Name).
			Where(tq.ID.Eq(id)).
			Updates(disposition)
		if err != nil {
			return err
		}
		if info.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (c *tenantsRepoImpl) FindOrCreateDispositionByName(name string) (*ldspec.DbDisposition, error) {
	var disposition *ldspec.DbDisposition
	err := c.transaction(func(q *tenantQuery.Query) error {
		tq := q.DbDisposition
		found, err := tq.Where(tq.Name.Eq(name)).Order(tq.ID).First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if found != nil {
			disposition = found
			return nil
		}
		// a new record on every attempt: Create sets the ID even when the
		// commit fails
		disposition = &ldspec.DbDisposition{Name: name}
		return tq.Create(disposition)
	})
	if err != nil {
		return nil, err
	}
	return disposition, nil
}
