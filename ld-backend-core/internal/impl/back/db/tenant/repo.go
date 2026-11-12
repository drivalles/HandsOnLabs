package ldtenant

import (
	"ld-backend/internal/impl/back/db/conn"
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"

	"gorm.io/gorm"
)

type tenantsRepoImpl struct {
	db       *gorm.DB
	tenantDb *tenantQuery.Query
}

func newTenantsRepo(db *gorm.DB, tenantDb *tenantQuery.Query) ldspec.TenantRepo {
	return &tenantsRepoImpl{
		db:       db,
		tenantDb: tenantDb,
	}
}

// transaction runs fn in a transaction with the queries bound to it, trying
// again when it conflicts with another; fn may run more than once.
func (c *tenantsRepoImpl) transaction(fn func(tq *tenantQuery.Query) error) error {
	return ldconn.Transaction(c.db, func(tx *gorm.DB) error {
		return fn(tenantQuery.Use(tx))
	})
}
