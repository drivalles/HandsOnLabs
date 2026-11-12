package ldtenant

import (
	"fmt"
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"
	"log/slog"
)

func (p *tenantsRepoImpl) DeleteStage(record *ldspec.DbPipelineStage) error {
	return p.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbPipelineStage.Unscoped().Delete(record)
		return err
	})
}

func (p *tenantsRepoImpl) SaveStage(record *ldspec.DbPipelineStage) error {
	return p.transaction(func(tq *tenantQuery.Query) error {
		return tq.DbPipelineStage.Save(record)
	})
}

func (c *tenantsRepoImpl) FindStageById(stageId uint) (*ldspec.DbPipelineStage, error) {
	return c.tenantDb.DbPipelineStage.
		Preload(c.tenantDb.DbPipelineStage.Pipeline).
		Where(c.tenantDb.DbPipelineStage.ID.Eq(stageId)).
		First()
}

func (c *tenantsRepoImpl) FindStagesByPipelineId(id uint) ([]*ldspec.DbPipelineStage, error) {
	return c.tenantDb.DbPipelineStage.Where(c.tenantDb.DbPipelineStage.PipelineId.Eq(id)).Find()
}

func (c *tenantsRepoImpl) DeleteStagesByPipelineId(id uint) error {
	return c.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbPipelineStage.Unscoped().
			Where(tq.DbPipelineStage.PipelineId.Eq(id)).Delete()
		return err
	})
}

func (c *tenantsRepoImpl) FindStageByGHLId(pipelineId uint, stage string) (*ldspec.DbPipelineStage, error) {
	tq := c.tenantDb
	return tq.DbPipelineStage.
		Where(tq.DbPipelineStage.PipelineId.Eq(pipelineId)).
		Where(tq.DbPipelineStage.GHLId.Eq(stage)).
		First()
}

func (c *tenantsRepoImpl) FindStagesByPipeline(record *ldspec.DbPipeline) []*ldspec.DbPipelineStage {
	tq := c.tenantDb
	stages, err := tq.DbPipelineStage.
		Preload(tq.DbPipelineStage.Pipeline).
		Where(tq.DbPipelineStage.PipelineId.Eq(record.ID)).
		Find()
	if err != nil {
		slog.Error(fmt.Sprintf("error getting stages for pipeline %d: %v", record.ID, err))
		return nil
	}

	return stages
}
