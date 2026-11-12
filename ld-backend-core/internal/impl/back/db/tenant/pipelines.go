package ldtenant

import (
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"
)

func (p *tenantsRepoImpl) DeletePipeline(record *ldspec.DbPipeline) error {
	return p.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbPipeline.Unscoped().Delete(record)
		return err
	})
}

func (p *tenantsRepoImpl) SavePipeline(record *ldspec.DbPipeline) error {
	return p.transaction(func(tq *tenantQuery.Query) error {
		return tq.DbPipeline.Save(record)
	})
}

func (c *tenantsRepoImpl) FindPipelines() ([]*ldspec.DbPipeline, error) {
	return c.tenantDb.DbPipeline.Find()
}

func (c *tenantsRepoImpl) FindPipeline(pipelineId string) (*ldspec.DbPipeline, error) {
	tq := c.tenantDb
	return tq.DbPipeline.
		Where(tq.DbPipeline.GHLId.Eq(pipelineId)).
		First()
}

func (c *tenantsRepoImpl) FindPipelineById(id uint) (*ldspec.DbPipeline, error) {
	tq := c.tenantDb
	return tq.DbPipeline.
		Where(tq.DbPipeline.ID.Eq(id)).
		First()
}
