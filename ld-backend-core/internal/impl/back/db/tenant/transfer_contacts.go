package ldtenant

import (
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"

	"gorm.io/gorm"
)

func (c *tenantsRepoImpl) FindTransferContacts() ([]*ldspec.DbTransferContact, error) {
	tCont := c.tenantDb.DbTransferContact
	return tCont.
		Order(tCont.Name).
		Find()
}

func (c *tenantsRepoImpl) SaveTransferContact(contact *ldspec.DbTransferContact) error {
	// isNew is decided before the transaction: Create sets the ID even when
	// the commit fails, and a retry must create the contact again.
	isNew := contact.ID == 0
	return c.transaction(func(q *tenantQuery.Query) error {
		tq := q.DbTransferContact
		if isNew {
			contact.ID = 0
			return tq.Create(contact)
		}
		info, err := tq.
			Select(tq.Name, tq.PhoneNumber).
			Where(tq.ID.Eq(contact.ID)).
			Updates(contact)
		if err != nil {
			return err
		}
		if info.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
