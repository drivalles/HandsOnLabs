package ldtenant

import (
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"
	"time"
)

func (c *tenantsRepoImpl) SaveCall(record *ldspec.DbCall) error {
	return c.transaction(func(tq *tenantQuery.Query) error {
		return tq.DbCall.Save(record)
	})
}

func (p *tenantsRepoImpl) DeleteCall(record *ldspec.DbCall) error {
	return p.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbCall.Unscoped().Delete(record)
		return err
	})
}

func (m *tenantsRepoImpl) FindCalls(ids []string) ([]*ldspec.DbCall, error) {
	return m.tenantDb.DbCall.
		Preload(m.tenantDb.DbCall.User).
		Preload(m.tenantDb.DbCall.Did).
		Preload(m.tenantDb.DbCall.Contact).
		Preload(m.tenantDb.DbCall.Disposition).
		Where(m.tenantDb.DbCall.GHLId.In(ids...)).
		Order(m.tenantDb.DbCall.DateTime.Desc()).
		Find()
}

func (m *tenantsRepoImpl) FindCallById(id uint) (*ldspec.DbCall, error) {
	return m.tenantDb.DbCall.
		Preload(m.tenantDb.DbCall.User).
		Preload(m.tenantDb.DbCall.Did).
		Preload(m.tenantDb.DbCall.Contact).
		Preload(m.tenantDb.DbCall.Disposition).
		Where(m.tenantDb.DbCall.ID.Eq(id)).
		First()
}

func (m *tenantsRepoImpl) FindCallsByIds(ids []uint) ([]*ldspec.DbCall, error) {
	return m.tenantDb.DbCall.
		Preload(m.tenantDb.DbCall.User).
		Preload(m.tenantDb.DbCall.Did).
		Preload(m.tenantDb.DbCall.Contact).
		Preload(m.tenantDb.DbCall.Disposition).
		Where(m.tenantDb.DbCall.ID.In(ids...)).
		Order(m.tenantDb.DbCall.DateTime.Desc()).
		Find()
}

func (m *tenantsRepoImpl) CountCalls() (int64, error) {
	return m.tenantDb.DbCall.Count()
}

func (m *tenantsRepoImpl) FindCall(id string) (*ldspec.DbCall, error) {
	return m.tenantDb.DbCall.
		Preload(m.tenantDb.DbCall.User).
		Preload(m.tenantDb.DbCall.Did).
		Preload(m.tenantDb.DbCall.Contact).
		Preload(m.tenantDb.DbCall.Disposition).
		Where(m.tenantDb.DbCall.GHLId.Eq(id)).
		First()
}

func (m *tenantsRepoImpl) FindCallBySIPSessionId(sessionId string) (*ldspec.DbCall, error) {
	return m.tenantDb.DbCall.
		Preload(m.tenantDb.DbCall.User).
		Preload(m.tenantDb.DbCall.Did).
		Preload(m.tenantDb.DbCall.Contact).
		Preload(m.tenantDb.DbCall.Disposition).
		Where(m.tenantDb.DbCall.SIPSessionId.Eq(sessionId)).
		First()
}

func (m *tenantsRepoImpl) FindLatestCallHistory(userId uint, limit int) ([]*ldspec.DbCall, error) {
	return m.tenantDb.DbCall.
		Preload(m.tenantDb.DbCall.User).
		Preload(m.tenantDb.DbCall.Did).
		Preload(m.tenantDb.DbCall.Contact).
		Preload(m.tenantDb.DbCall.Disposition).
		Where(m.tenantDb.DbCall.UserId.Eq(userId)).
		Order(m.tenantDb.DbCall.DateTime.Desc()).
		Limit(limit).
		Find()
}

func (m *tenantsRepoImpl) FindCallHistory(filter *ldspec.DbCallHistoryFilter) ([]*ldspec.DbCall, int64, error) {
	tq := m.tenantDb.DbCall
	do := tq.
		Preload(tq.User).
		Preload(tq.Did).
		Preload(tq.Contact).
		Preload(tq.Disposition)
	if filter.From != nil {
		do = do.Where(tq.DateTime.Gte(*filter.From))
	}
	if filter.To != nil {
		do = do.Where(tq.DateTime.Lt(*filter.To))
	}
	if filter.CallStatus != nil {
		do = do.Where(tq.CallStatus.Eq(*filter.CallStatus))
	}
	if filter.Outbound != nil {
		do = do.Where(tq.Outbound.Is(*filter.Outbound))
	}
	// the GHL ids of the call are set even when its user or contact is not
	// imported yet, as for the calls imported from GHL
	if len(filter.UserGHLIds) > 0 {
		do = do.Where(tq.GHLUserId.In(filter.UserGHLIds...))
	}
	if len(filter.ContactGHLIds) > 0 {
		do = do.Where(tq.GHLContactId.In(filter.ContactGHLIds...))
	}
	// the id breaks the ties of the date, so the pages do not overlap
	return do.
		Order(tq.DateTime.Desc(), tq.ID.Desc()).
		FindByPage(filter.Offset, filter.Limit)
}

func (m *tenantsRepoImpl) FindCallsByContact(contactId string) ([]*ldspec.DbCall, error) {
	dbContact, err := m.tenantDb.DbContact.Where(m.tenantDb.DbContact.GHLId.Eq(contactId)).First()
	if err != nil {
		return nil, err
	}
	return m.tenantDb.DbCall.
		Preload(m.tenantDb.DbCall.User).
		Preload(m.tenantDb.DbCall.Did).
		Preload(m.tenantDb.DbCall.Contact).
		Preload(m.tenantDb.DbCall.Disposition).
		Where(m.tenantDb.DbCall.ContactId.Eq(dbContact.ID)).
		Order(m.tenantDb.DbCall.DateTime.Desc()).
		Find()
}

func (m *tenantsRepoImpl) FindCallsByDate(date time.Time) ([]*ldspec.DbCall, error) {
	return m.tenantDb.DbCall.
		Preload(m.tenantDb.DbCall.User).
		Preload(m.tenantDb.DbCall.Did).
		Preload(m.tenantDb.DbCall.Contact).
		Preload(m.tenantDb.DbCall.Disposition).
		Where(m.tenantDb.DbCall.DateTime.Gte(date)).
		Where(m.tenantDb.DbCall.DateTime.Lte(date.Add(24 * time.Hour))).
		Where(m.tenantDb.DbCall.ContactId.IsNotNull()).
		Order(m.tenantDb.DbCall.DateTime.Desc()).
		Find()
}

func (m *tenantsRepoImpl) DeleteCallById(id uint) error {
	return m.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbCall.
			Unscoped().
			Where(tq.DbCall.ID.Eq(id)).
			Delete()
		return err
	})
}
