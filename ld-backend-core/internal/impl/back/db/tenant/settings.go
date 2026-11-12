package ldtenant

import (
	"errors"
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"

	"gorm.io/gorm"
)

func (p *tenantsRepoImpl) GetNewLeadsStage() (*ldspec.DbPipelineStage, error) {
	settings, err := p.tenantDb.DbTenantSettings.
		Preload(p.tenantDb.DbTenantSettings.NewLeadsStage).
		Preload(p.tenantDb.DbTenantSettings.NewLeadsStage.Pipeline).
		Where(p.tenantDb.DbTenantSettings.PK.Eq("default")).
		First()
	if err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return nil, nil
		}
		return nil, err
	}
	if settings.NewLeadsStage != nil {
		return settings.NewLeadsStage, nil
	}
	return nil, nil
}

func (p *tenantsRepoImpl) SetNewLeadsStage(stageId uint) error {
	return p.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbTenantSettings.
			Where(tq.DbTenantSettings.PK.Eq("default")).
			First()
		if err != nil {
			if errors.Is(gorm.ErrRecordNotFound, err) {
				return tq.DbTenantSettings.Save(&ldspec.DbTenantSettings{
					PK:              "default",
					NewLeadsStageId: &stageId,
				})
			}
			return err
		}
		_, err = tq.DbTenantSettings.
			Where(tq.DbTenantSettings.PK.Eq("default")).
			Update(tq.DbTenantSettings.NewLeadsStageId, &stageId)
		return err
	})
}

func (p *tenantsRepoImpl) GetDispositionField() (string, error) {
	settings, err := p.tenantDb.DbTenantSettings.
		Where(p.tenantDb.DbTenantSettings.PK.Eq("default")).
		First()
	if err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return "", nil
		}
		return "", err
	}
	if settings.DispositionFieldGHLId != nil {
		return *settings.DispositionFieldGHLId, nil
	}
	return "", nil
}

func (p *tenantsRepoImpl) SetDispositionField(ghlId string) error {
	return p.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbTenantSettings.
			Where(tq.DbTenantSettings.PK.Eq("default")).
			First()
		if err != nil {
			if errors.Is(gorm.ErrRecordNotFound, err) {
				return tq.DbTenantSettings.Save(&ldspec.DbTenantSettings{
					PK:                    "default",
					DispositionFieldGHLId: &ghlId,
				})
			}
			return err
		}
		_, err = tq.DbTenantSettings.
			Where(tq.DbTenantSettings.PK.Eq("default")).
			Update(tq.DbTenantSettings.DispositionFieldGHLId, &ghlId)
		return err
	})
}
