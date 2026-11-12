package ldtenant

import (
	"errors"
	"fmt"
	tenantQuery "ld-backend/internal/impl/back/db/tenant/query"
	lddata "ld-backend/internal/impl/core/data"
	"ld-backend/internal/spec"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

func (m *tenantsRepoImpl) UpdateDidCallStats(phoneNumber string) {
	if phoneNumber == "" {
		return
	}
	err := m.transaction(func(tq *tenantQuery.Query) error {
		did, err := tq.DbDid.Where(tq.DbDid.DidNumber.Eq(phoneNumber)).First()
		if err != nil {
			return fmt.Errorf("error getting did %s: %w", phoneNumber, err)
		}
		count, err := tq.DbCall.
			Where(tq.DbCall.DidNumber.Eq(phoneNumber), tq.DbCall.DateTime.Gt(time.Now().Add(-10*24*time.Hour))).
			Count()
		if err != nil {
			return fmt.Errorf("error getting did call count: %w", err)
		}
		did.NumberOfCalls = int16(count)
		lastCall, err := tq.DbCall.
			Where(tq.DbCall.DidNumber.Eq(phoneNumber)).Order(tq.DbCall.DateTime.Desc()).
			Limit(1).
			First()
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("error getting last did call: %w", err)
			}
		} else {
			did.LastCallId = &lastCall.ID
		}
		if err = tq.DbDid.Save(did); err != nil {
			return fmt.Errorf("error saving did call stats: %w", err)
		}
		return nil
	})
	if err != nil {
		slog.Error(fmt.Sprintf("error updating did call stats: %v", err), "module", "impl/back/info/imports")
	}
}

func (c *tenantsRepoImpl) FindOrCreateDid(from *string, enabled bool) (*ldspec.DbDid, error) {
	if from == nil {
		return nil, fmt.Errorf("nil from number")
	}

	number, err := lddata.ValidatePhoneNumber(*from)
	if number == "" || err != nil {
		slog.Warn(fmt.Sprintf("Skipping DID %s, too long or invalid", *from))
		return nil, fmt.Errorf("invalid did number %s", *from)
	}

	var dbDid *ldspec.DbDid
	err = c.transaction(func(tq *tenantQuery.Query) error {
		var err error
		dbDid, err = tq.DbDid.
			Where(tq.DbDid.DidNumber.Eq(number)).
			First()
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("error getting did for number %s: %w", *from, err)
		}
		dbDid = &ldspec.DbDid{
			DidNumber: number,
			Enabled:   &enabled,
		}
		if err = tq.DbDid.Save(dbDid); err != nil {
			return fmt.Errorf("error saving did for number %s: %w", *from, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dbDid, nil
}

func (c *tenantsRepoImpl) FindEnabledDids() ([]*ldspec.DbDid, error) {
	return c.tenantDb.DbDid.
		Where(c.tenantDb.DbDid.Enabled.Eq(true)).
		Find()
}

func (c *tenantsRepoImpl) FindAllDids() ([]*ldspec.DbDid, error) {
	return c.tenantDb.DbDid.
		Find()
}

func (c *tenantsRepoImpl) FindDid(number string) (*ldspec.DbDid, error) {
	tq := c.tenantDb
	return tq.DbDid.
		Where(tq.DbDid.DidNumber.Eq(number)).
		First()
}

func (il *tenantsRepoImpl) SaveDid(did *ldspec.DbDid) error {
	return il.transaction(func(tq *tenantQuery.Query) error {
		return tq.DbDid.Save(did)
	})
}

// LockDid releases the lock of the did when it is older than 10 minutes and
// locks it, both in the same transaction.
func (il *tenantsRepoImpl) LockDid(number string, lockId string, userId uint) (bool, error) {
	locked := false
	err := il.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbDid.
			Where(tq.DbDid.DidNumber.Eq(number)).
			Where(tq.DbDid.LockTs.IsNotNull(), tq.DbDid.LockTs.Lt(time.Now().Add(-10*time.Minute))).
			UpdateSimple(tq.DbDid.LockId.Null(), tq.DbDid.LockTs.Null())
		if err != nil {
			return err
		}

		simple, err := tq.DbDid.
			Where(tq.DbDid.DidNumber.Eq(number)).
			Where(tq.DbDid.Where(tq.DbDid.LockId.IsNull()).Or(tq.DbDid.LockId.Eq(lockId))).
			UpdateSimple(tq.DbDid.LockId.Value(lockId),
				tq.DbDid.LockTs.Value(time.Now()),
				tq.DbDid.LockUserId.Value(userId))
		if err != nil {
			return err
		}
		locked = simple.RowsAffected == 1
		return nil
	})
	if err != nil {
		return false, err
	}
	return locked, nil
}

func (il *tenantsRepoImpl) UnlockDid(number string, lockId string) (bool, error) {
	unlocked := false
	err := il.transaction(func(tq *tenantQuery.Query) error {
		simple, err := tq.DbDid.
			Where(tq.DbDid.DidNumber.Eq(number)).
			Where(tq.DbDid.LockId.Eq(lockId)).
			UpdateSimple(tq.DbDid.LockId.Null(), tq.DbDid.LockTs.Null(), tq.DbDid.LockUserId.Null())
		if err != nil {
			return err
		}
		unlocked = simple.RowsAffected == 1
		return nil
	})
	if err != nil {
		return false, err
	}
	return unlocked, nil
}

// FindDidHealthReports returns the health reports of the DID, the newest
// first.
func (c *tenantsRepoImpl) FindDidHealthReports(didId uint) ([]*ldspec.DbDidHealthReport, error) {
	tq := c.tenantDb
	return tq.DbDidHealthReport.
		Where(tq.DbDidHealthReport.DidId.Eq(didId)).
		Order(tq.DbDidHealthReport.CreatedAt.Desc(), tq.DbDidHealthReport.ID.Desc()).
		Find()
}

func (c *tenantsRepoImpl) SaveDidHealthReport(report *ldspec.DbDidHealthReport) error {
	return c.transaction(func(tq *tenantQuery.Query) error {
		return tq.DbDidHealthReport.Save(report)
	})
}
