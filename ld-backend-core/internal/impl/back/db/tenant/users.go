package ldtenant

import (
	"errors"
	"fmt"
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"

	"gorm.io/gorm"
)

func (c *tenantsRepoImpl) FindUser(id *string) (*ldspec.DbUser, error) {
	if id == nil {
		return nil, fmt.Errorf("nil user ID")
	}

	tq := c.tenantDb
	return tq.DbUser.
		Where(tq.DbUser.GHLId.Eq(*id)).
		First()
}

func (c *tenantsRepoImpl) FindOrCreateUser(id *string, name string) (*ldspec.DbUser, error) {
	if id == nil {
		return nil, fmt.Errorf("nil user ID")
	}
	var ldUser *ldspec.DbUser
	err := c.transaction(func(tq *tenantQuery.Query) error {
		var err error
		ldUser, err = tq.DbUser.
			Where(tq.DbUser.GHLId.Eq(*id)).
			First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("error finding or creating user for %s: %w", *id, err)
		}
		if ldUser == nil {
			ldUser = &ldspec.DbUser{
				GHLId:    id,
				UserName: &name,
			}
			if err = tq.DbUser.Save(ldUser); err != nil {
				return fmt.Errorf("error saving new user for %s: %w", *id, err)
			}
			return nil
		}
		if name != "" && (ldUser.UserName == nil || *ldUser.UserName == "") {
			ldUser.UserName = &name
			return tq.DbUser.Save(ldUser)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ldUser, nil
}

func (c *tenantsRepoImpl) SaveUser(user *ldspec.DbUser) error {
	return c.transaction(func(tq *tenantQuery.Query) error {
		return tq.DbUser.Save(user)
	})
}

// UpdateUserAccess sets the access of the user with the id, updating only
// that column.
func (c *tenantsRepoImpl) UpdateUserAccess(id uint, access ldspec.LDUserAccess) error {
	return c.transaction(func(tq *tenantQuery.Query) error {
		info, err := tq.DbUser.
			Where(tq.DbUser.ID.Eq(id)).
			Update(tq.DbUser.LDAccessType, string(access))
		if err != nil {
			return err
		}
		if info.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (c *tenantsRepoImpl) FindUsers() ([]*ldspec.DbUser, error) {
	return c.tenantDb.DbUser.Find()
}
