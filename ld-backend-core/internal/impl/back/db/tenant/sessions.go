package ldtenant

import (
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	"ld-backend/internal/spec"
)

func (c *tenantsRepoImpl) FindDialerSessionBySessionId(sessionId string) (*ldspec.DbDialerSession, error) {
	return c.tenantDb.DbDialerSession.
		Where(c.tenantDb.DbDialerSession.SessionId.Eq(sessionId)).
		First()
}

func (c *tenantsRepoImpl) FindDialerSessions(userId *uint) ([]*ldspec.DbDialerSession, error) {
	session, err := c.tenantDb.DbDialerSession.
		Preload(c.tenantDb.DbDialerSession.User).
		Preload(c.tenantDb.DbDialerSession.LastCall).
		Preload(c.tenantDb.DbDialerSession.LastCall.Contact).
		Preload(c.tenantDb.DbDialerSession.LastCall.Did).
		Preload(c.tenantDb.DbDialerSession.Pipeline).
		Where(c.tenantDb.DbDialerSession.UserId.Eq(*userId)).
		Order(c.tenantDb.DbDialerSession.UpdatedAt.Desc()).
		Find()
	return session, err
}

func (c *tenantsRepoImpl) FindOpenDialerSessions(locationId string) ([]*ldspec.DbDialerSession, error) {
	session, err := c.tenantDb.DbDialerSession.
		Preload(c.tenantDb.DbDialerSession.User).
		Preload(c.tenantDb.DbDialerSession.LastCall).
		Preload(c.tenantDb.DbDialerSession.LastCall.Contact).
		Preload(c.tenantDb.DbDialerSession.LastCall.Did).
		Preload(c.tenantDb.DbDialerSession.Pipeline).
		Where(c.tenantDb.DbDialerSession.CallerStatus.IsNotNull()).
		Where(c.tenantDb.DbDialerSession.LocationId.Eq(locationId)).
		Order(c.tenantDb.DbDialerSession.UpdatedAt.Desc()).
		Find()
	return session, err
}

func (c *tenantsRepoImpl) SaveDialerSession(session *ldspec.DbDialerSession) error {
	return c.transaction(func(tq *tenantQuery.Query) error {
		return tq.DbDialerSession.Save(session)
	})
}

func (c *tenantsRepoImpl) RemoveOldDialerSessions(nodeId string) error {
	return c.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbDialerSession.
			Unscoped().
			Where(tq.DbDialerSession.NodeId.IsNull()).Or(tq.DbDialerSession.NodeId.Eq("")).Or(tq.DbDialerSession.NodeId.Eq(nodeId)).
			Delete()
		return err
	})
}
