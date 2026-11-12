package ldtenant

import (
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"

	"gorm.io/gorm"
)

func (c *tenantsRepoImpl) FindCallScripts() ([]*ldspec.DbCallScript, error) {
	return c.tenantDb.DbCallScript.
		Order(c.tenantDb.DbCallScript.Name).
		Find()
}

func (c *tenantsRepoImpl) SaveCallScript(script *ldspec.DbCallScript) error {
	// isNew is decided before the transaction: Create sets the ID even when
	// the commit fails, and a retry must create the script again.
	isNew := script.ID == 0
	return c.transaction(func(q *tenantQuery.Query) error {
		tq := q.DbCallScript
		if isNew {
			script.ID = 0
			return tq.Create(script)
		}
		info, err := tq.
			Select(tq.Name, tq.Text).
			Where(tq.ID.Eq(script.ID)).
			Updates(script)
		if err != nil {
			return err
		}
		if info.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
