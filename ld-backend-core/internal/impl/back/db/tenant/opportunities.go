package ldtenant

import (
	"fmt"
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"
	"time"
)

func (c *tenantsRepoImpl) DeleteOpportunity(record *ldspec.DbOpportunity) error {
	return c.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbOpportunity.Unscoped().Delete(record)
		return err
	})
}

func (c *tenantsRepoImpl) SaveOpportunity(record *ldspec.DbOpportunity) error {
	return c.transaction(func(tq *tenantQuery.Query) error {
		return tq.DbOpportunity.Save(record)
	})
}

func (m *tenantsRepoImpl) FindOpportunities(pipeline *ldspec.DbPipeline, stage *ldspec.DbPipelineStage, assignedTo *ldspec.DbUser) ([]*ldspec.DbOpportunity, error) {
	dbbOpp := m.tenantDb.DbOpportunity
	q := dbbOpp.
		Preload(dbbOpp.Contact).
		Preload(dbbOpp.Pipeline).
		Preload(dbbOpp.Stage).
		Preload(dbbOpp.AssignedTo).
		Where(dbbOpp.PipelineId.Eq(pipeline.ID), dbbOpp.StageId.Eq(stage.ID))
	if assignedTo != nil {
		q = q.Where(dbbOpp.AssignedToId.Eq(assignedTo.ID))
	}
	return q.Find()
}

func (m *tenantsRepoImpl) FindOpportunitiesByContact(contactId *ldspec.DbContact) ([]*ldspec.DbOpportunity, error) {
	dbbOpp := m.tenantDb.DbOpportunity
	q := dbbOpp.
		Preload(dbbOpp.Contact).
		Preload(dbbOpp.Pipeline).
		Preload(dbbOpp.Stage).
		Preload(dbbOpp.AssignedTo).
		Where(dbbOpp.ContactId.Eq(contactId.ID))
	return q.Find()
}

func (m *tenantsRepoImpl) FindOpportunitiesByPipeline(pipeline *ldspec.DbPipeline) ([]*ldspec.DbOpportunity, error) {
	dbbOpp := m.tenantDb.DbOpportunity
	q := dbbOpp.
		Preload(dbbOpp.Contact).
		Preload(dbbOpp.Pipeline).
		Preload(dbbOpp.Stage).
		Preload(dbbOpp.AssignedTo).
		Where(dbbOpp.PipelineId.Eq(pipeline.ID))
	return q.Find()
}

func (m *tenantsRepoImpl) FindOpportunityByGHLId(ghlId string) (*ldspec.DbOpportunity, error) {
	dbbOpp := m.tenantDb.DbOpportunity
	q := dbbOpp.
		Preload(dbbOpp.Contact).
		Preload(dbbOpp.Pipeline).
		Preload(dbbOpp.Stage).
		Preload(dbbOpp.AssignedTo).
		Where(dbbOpp.GHLId.Eq(ghlId))
	return q.First()
}

func (m *tenantsRepoImpl) FindOpportunity(pipeline *ldspec.DbPipeline, contact *ldspec.DbContact) (*ldspec.DbOpportunity, error) {
	dbbOpp := m.tenantDb.DbOpportunity
	q := dbbOpp.
		Preload(dbbOpp.Contact).
		Preload(dbbOpp.Pipeline).
		Preload(dbbOpp.Stage).
		Preload(dbbOpp.AssignedTo).
		Where(dbbOpp.PipelineId.Eq(pipeline.ID), dbbOpp.ContactId.Eq(contact.ID))
	return q.First()
}

func (m *tenantsRepoImpl) FindNewOpportunities() ([]*ldspec.DbOpportunity, error) {
	date := time.Now().Add(-24 * time.Hour)
	contacts, err := m.tenantDb.DbContact.
		Where(m.tenantDb.DbContact.DateRegistered.Date().Gte(date)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("error getting contacts: %v", err)
	}
	ids := getContactIds(contacts)

	leadsStage, err := m.GetNewLeadsStage()
	if err != nil {
		return nil, err
	}

	if leadsStage == nil || leadsStage.Pipeline == nil {
		return []*ldspec.DbOpportunity{}, nil
	}

	dbbOpp := m.tenantDb.DbOpportunity
	q := dbbOpp.
		Preload(dbbOpp.Contact).
		Preload(dbbOpp.Pipeline).
		Preload(dbbOpp.Stage).
		Preload(dbbOpp.AssignedTo).
		Where(dbbOpp.PipelineId.Eq(leadsStage.Pipeline.ID), dbbOpp.StageId.Eq(leadsStage.ID)).
		Where(dbbOpp.ContactId.In(ids...))
	return q.Find()
}

func (m *tenantsRepoImpl) FindCallbackOpportunities(callbackUser *ldspec.DbUser) ([]*ldspec.DbOpportunity, error) {
	contactsQuery := m.tenantDb.DbContact.
		Where(m.tenantDb.DbContact.CallbackTime.Lte(time.Now()))
	if callbackUser != nil {
		contactsQuery = contactsQuery.Where(m.tenantDb.DbContact.CallbackUserId.Eq(callbackUser.ID))
	}
	contacts, err := contactsQuery.Find()
	if err != nil {
		return nil, fmt.Errorf("error getting contacts: %v", err)
	}
	if len(contacts) == 0 {
		return []*ldspec.DbOpportunity{}, nil
	}

	dbbOpp := m.tenantDb.DbOpportunity
	q := dbbOpp.
		Preload(dbbOpp.Contact).
		Preload(dbbOpp.Contact.CallbackUser).
		Preload(dbbOpp.Pipeline).
		Preload(dbbOpp.Stage).
		Preload(dbbOpp.AssignedTo).
		Where(dbbOpp.ContactId.In(getContactIds(contacts)...))
	return q.Find()
}

func getContactIds(contacts []*ldspec.DbContact) []uint {
	ids := make([]uint, 0, len(contacts))
	for _, contact := range contacts {
		ids = append(ids, contact.ID)
	}
	return ids
}
