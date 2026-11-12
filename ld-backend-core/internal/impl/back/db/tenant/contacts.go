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

func (c *tenantsRepoImpl) FindContactByGHLId(ghlId string) (*ldspec.DbContact, error) {
	tq := c.tenantDb
	return tq.DbContact.
		Where(tq.DbContact.GHLId.Eq(ghlId)).
		First()
}

func (c *tenantsRepoImpl) FindContactById(id uint) (*ldspec.DbContact, error) {
	tq := c.tenantDb
	return tq.DbContact.
		Where(tq.DbContact.ID.Eq(id)).
		First()
}

// FindContactByPhoneNumber finds the newest contact with the phone number; the
// numbers are stored as GHL sends them, so the US formats +1XXXXXXXXXX,
// 1XXXXXXXXXX and XXXXXXXXXX are matched.
func (c *tenantsRepoImpl) FindContactByPhoneNumber(phoneNumber string) (*ldspec.DbContact, error) {
	number, err := lddata.ValidatePhoneNumber(phoneNumber)
	if err != nil {
		return nil, err
	}
	tq := c.tenantDb
	return tq.DbContact.
		Where(tq.DbContact.PhoneNumber.In("+"+number, number, number[1:])).
		Order(tq.DbContact.ID.Desc()).
		First()
}

type ContactsNoCalls struct {
	Id            uint
	NumberOfCalls int16
}

func (c *tenantsRepoImpl) UpdateContactsNoCalls() {
	tq := c.tenantDb
	result := []ContactsNoCalls{}
	err := tq.DbContact.
		Select(tq.DbContact.ID, tq.DbCall.ID.Count().As("number_of_calls")).
		LeftJoin(tq.DbCall, tq.DbCall.ContactId.EqCol(tq.DbContact.ID)).
		Group(tq.DbContact.ID).
		Scan(&result)
	if err != nil {
		slog.Error(fmt.Sprintf("error counting calls per contact: %v", err))
		return
	}
	err = c.transaction(func(tq *tenantQuery.Query) error {
		for _, r := range result {
			_, err := tq.DbContact.Where(tq.DbContact.ID.Eq(r.Id)).UpdateSimple(tq.DbContact.NumberOfCalls.Value(r.NumberOfCalls))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		slog.Error(fmt.Sprintf("error updating calls per contact: %v", err))
	}
}

func (c *tenantsRepoImpl) FindContactTags(contactId uint) []string {
	tq := c.tenantDb
	tags, err := tq.DbContactTag.
		Where(tq.DbContactTag.ContactId.Eq(contactId)).
		Find()
	if err != nil {
		slog.Error(fmt.Sprintf("error getting tags for contact %d: %v", contactId, err))
		return nil
	}

	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		result = append(result, tag.Tags)
	}
	return result
}

func (il *tenantsRepoImpl) ReleaseAllLocks(nodeId string) {
	err := il.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbContact.
			Where(tq.DbContact.LockId.Eq(nodeId)).
			UpdateSimple(tq.DbContact.LockId.Null(), tq.DbContact.LockTs.Null(), tq.DbContact.LockUserId.Null())
		if err != nil {
			return fmt.Errorf("error releasing contacts: %w", err)
		}

		_, err = tq.DbDid.
			Where(tq.DbDid.LockId.Eq(nodeId)).
			UpdateSimple(tq.DbDid.LockId.Null(), tq.DbDid.LockTs.Null(), tq.DbDid.LockUserId.Null())
		if err != nil {
			return fmt.Errorf("error releasing dids: %w", err)
		}
		return nil
	})
	if err != nil {
		slog.Error(fmt.Sprintf("error releasing locks for node %s: %v", nodeId, err))
	}
}

func (il *tenantsRepoImpl) LockMaintenance(nodeId string) {
	tq := il.tenantDb

	contacts, err := tq.DbContact.
		Preload(tq.DbContact.LastCall).
		Preload(tq.DbContact.LockUser).
		Where(tq.DbContact.LockTs.IsNotNull(), tq.DbContact.LockTs.Lt(time.Now().Add(-2*time.Hour))).
		Find()
	if err != nil {
		slog.Error(fmt.Sprintf("error getting contacts to unlock: %v", err))
		return
	}

	for _, contact := range contacts {
		if il.isCompleted(contact.LastCall) {
			slog.Info(fmt.Sprintf("unlocking contact %s that was locked for more than 2 hours", contact.PhoneNumber))
			err = il.transaction(func(tq *tenantQuery.Query) error {
				_, err := tq.DbContact.
					Where(tq.DbContact.ID.Eq(contact.ID)).
					UpdateSimple(tq.DbContact.LockId.Null(), tq.DbContact.LockTs.Null(), tq.DbContact.LockUserId.Null())
				return err
			})
			if err != nil {
				slog.Error(fmt.Sprintf("error unlocking contact %d: %v", contact.ID, err))
			}
		}
	}

	dids, err := tq.DbDid.
		Preload(tq.DbDid.LastCall).
		Preload(tq.DbDid.LockUser).
		Where(tq.DbDid.LockTs.IsNotNull(), tq.DbDid.LockTs.Lt(time.Now().Add(-2*time.Hour))).
		Find()
	if err != nil {
		slog.Error(fmt.Sprintf("error getting contacts to unlock: %v", err))
		return
	}

	for _, did := range dids {
		if il.isCompleted(did.LastCall) {
			slog.Info(fmt.Sprintf("unlocking did %s that was locked for more than 2 hours", did.DidNumber))
			err = il.transaction(func(tq *tenantQuery.Query) error {
				_, err := tq.DbDid.
					Where(tq.DbDid.ID.Eq(did.ID)).
					UpdateSimple(tq.DbDid.LockId.Null(), tq.DbDid.LockTs.Null(), tq.DbDid.LockUserId.Null())
				return err
			})
			if err != nil {
				slog.Error(fmt.Sprintf("error unlocking did %s: %v", did.DidNumber, err))
			}
		}
	}
}

func (il *tenantsRepoImpl) LockContact(contactId string, lockId string, userId uint) (bool, error) {
	locked := false
	err := il.transaction(func(tq *tenantQuery.Query) error {
		simple, err := tq.DbContact.
			Where(tq.DbContact.GHLId.Eq(contactId)).
			Where(tq.DbContact.Where(tq.DbContact.LockId.IsNull()).Or(tq.DbContact.LockId.Eq(lockId))).
			UpdateSimple(tq.DbContact.LockId.Value(lockId),
				tq.DbContact.LockTs.Value(time.Now()),
				tq.DbContact.LastCallAttemptTime.Value(time.Now()),
				tq.DbContact.LockUserId.Value(userId))
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

func (il *tenantsRepoImpl) UnlockContact(contactId string, lockId string) (bool, error) {
	unlocked := false
	err := il.transaction(func(tq *tenantQuery.Query) error {
		simple, err := tq.DbContact.
			Where(tq.DbContact.GHLId.Eq(contactId)).
			Where(tq.DbContact.LockId.Eq(lockId)).
			UpdateSimple(tq.DbContact.LockId.Null(), tq.DbContact.LockTs.Null(), tq.DbContact.LockUserId.Null())
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

func (il *tenantsRepoImpl) ClearContactCallback(contactId string) error {
	return il.transaction(func(tq *tenantQuery.Query) error {
		_, err := tq.DbContact.
			Where(tq.DbContact.GHLId.Eq(contactId), tq.DbContact.CallbackTime.IsNotNull()).
			UpdateSimple(tq.DbContact.CallbackTime.Null(), tq.DbContact.CallbackUserId.Null())
		return err
	})
}

func (il *tenantsRepoImpl) UpdateContactCallback(contactId string, userId uint, callbackTime time.Time) error {
	return il.transaction(func(tq *tenantQuery.Query) error {
		info, err := tq.DbContact.
			Where(tq.DbContact.GHLId.Eq(contactId)).
			UpdateSimple(tq.DbContact.CallbackTime.Value(callbackTime), tq.DbContact.CallbackUserId.Value(userId))
		if err != nil {
			return err
		}
		if info.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (c *tenantsRepoImpl) SaveContact(contact *ldspec.DbContact) error {
	return c.transaction(func(tq *tenantQuery.Query) error {
		return tq.DbContact.Save(contact)
	})
}

func (c *tenantsRepoImpl) CountAbandonedLeads() (int64, error) {
	return c.tenantDb.DbContact.
		Where(c.tenantDb.DbContact.DateRegistered.Lt(time.Now().Add(-1 * time.Hour))).
		Where(c.tenantDb.DbContact.NumberOfCalls.Eq(0)).
		Count()
}

func (m *tenantsRepoImpl) isCompleted(call *ldspec.DbCall) bool {
	if call == nil {
		return true
	}
	if call.CallStatus == nil {
		return false
	}
	if *call.CallStatus == "completed" || *call.CallStatus == "cmp" {
		return true
	}
	return false
}

func (m *tenantsRepoImpl) UpdateContactCallStats(contactId *string) {
	if contactId == nil || *contactId == "" {
		return
	}
	err := m.transaction(func(tq *tenantQuery.Query) error {
		contact, err := tq.DbContact.
			Where(tq.DbContact.GHLId.Eq(*contactId)).
			First()
		if err != nil {
			return fmt.Errorf("error getting contact: %w", err)
		}
		count, err := tq.DbCall.
			Where(tq.DbCall.GHLContactId.Eq(*contactId)).
			Count()
		if err != nil {
			return fmt.Errorf("error getting contact call count: %w", err)
		}
		contact.NumberOfCalls = int16(count)
		if count > 0 {
			lastCall, err := tq.DbCall.
				Where(tq.DbCall.GHLContactId.Eq(*contactId)).Order(tq.DbCall.DateTime.Desc()).Limit(1).
				First()
			if err != nil {
				return fmt.Errorf("error getting last contact call: %w", err)
			}
			contact.LastCallId = &lastCall.ID
			contact.LastCallTime = lastCall.DateTime
		}
		if err = tq.DbContact.Save(contact); err != nil {
			return fmt.Errorf("error saving contact call stats: %w", err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		slog.Error(fmt.Sprintf("error updating contact call stats: %v", err), "module", "impl/back/db/repos")
	}
}
