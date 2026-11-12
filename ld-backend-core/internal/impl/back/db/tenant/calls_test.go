package ldtenant_test

import (
	ldconn "ld-backend/internal/impl/back/db/conn"
	ldtenant "ld-backend/internal/impl/back/db/tenant"
	ldutils "ld-backend/internal/impl/core/utils"
	ldspec "ld-backend/internal/spec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testSaveCall(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates and updates a call", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		call := saveCall(t, repo, "call-1", time.Now(), nil)
		require.NotZero(t, call.ID)

		call.CallStatus = ldutils.Ptr("completed")
		require.NoError(t, repo.SaveCall(call))

		found, err := repo.FindCall("call-1")
		require.NoError(t, err)
		assert.Equal(t, call.ID, found.ID)
		assert.Equal(t, "completed", *found.CallStatus)
		count, err := repo.CountCalls()
		require.NoError(t, err)
		assert.EqualValues(t, 1, count)
	})

	t.Run("a conflicting transaction is tried again", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		creates := ldconn.ConflictOnCreate(t, repo.Db(), "calls", 1)
		call := saveCall(t, repo, "call-1", time.Now(), nil)

		assert.EqualValues(t, 2, creates.Load())
		found, err := repo.FindCall("call-1")
		require.NoError(t, err)
		assert.Equal(t, call.ID, found.ID)
		count, err := repo.CountCalls()
		require.NoError(t, err)
		assert.EqualValues(t, 1, count)
	})

	t.Run("duplicated GHL id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		saveCall(t, repo, "call-1", time.Now(), nil)
		err := repo.SaveCall(&ldspec.DbCall{GHLId: ldutils.Ptr("call-1"), PhoneNumber: "+15550000002"})
		assert.Error(t, err)
	})
}

func testDeleteCall(t *testing.T, d ldconn.TestDriver) {
	t.Run("permanently deletes the call", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		call := saveCall(t, repo, "call-1", time.Now(), nil)
		saveCall(t, repo, "call-2", time.Now(), nil)

		require.NoError(t, repo.DeleteCall(call))
		_, err := repo.FindCall("call-1")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		var count int64
		require.NoError(t, repo.Db().Unscoped().Model(&ldspec.DbCall{}).Count(&count).Error)
		assert.EqualValues(t, 1, count, "the call is not soft deleted")
	})

	t.Run("permanently deletes the call by id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		call := saveCall(t, repo, "call-1", time.Now(), nil)
		saveCall(t, repo, "call-2", time.Now(), nil)

		require.NoError(t, repo.DeleteCallById(call.ID))
		_, err := repo.FindCallById(call.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		var count int64
		require.NoError(t, repo.Db().Unscoped().Model(&ldspec.DbCall{}).Count(&count).Error)
		assert.EqualValues(t, 1, count, "the call is not soft deleted")
	})

	t.Run("unknown id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		saveCall(t, repo, "call-1", time.Now(), nil)
		require.NoError(t, repo.DeleteCallById(999))
		count, err := repo.CountCalls()
		require.NoError(t, err)
		assert.EqualValues(t, 1, count)
	})
}

func testFindCall(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	user := saveUser(t, repo, "user-1", "User 1")
	contact := saveContact(t, repo, "contact-1", "+15550000001")
	did := saveDid(t, repo, "15550000002", true)
	disposition := &ldspec.DbDisposition{Name: "Interested"}
	require.NoError(t, repo.SaveDisposition(disposition))
	call := saveCall(t, repo, "call-1", time.Now(), func(call *ldspec.DbCall) {
		call.SIPSessionId = ldutils.Ptr("sip-1")
		call.UserId = &user.ID
		call.ContactId = &contact.ID
		call.DidId = &did.ID
		call.DispositionId = &disposition.ID
	})
	saveCall(t, repo, "call-2", time.Now(), nil)

	assertRelations := func(t *testing.T, found *ldspec.DbCall) {
		t.Helper()
		assert.Equal(t, call.ID, found.ID)
		require.NotNil(t, found.User)
		assert.Equal(t, "user-1", *found.User.GHLId)
		require.NotNil(t, found.Contact)
		assert.Equal(t, "contact-1", found.Contact.GHLId)
		require.NotNil(t, found.Did)
		assert.Equal(t, "15550000002", found.Did.DidNumber)
		require.NotNil(t, found.Disposition)
		assert.Equal(t, "Interested", found.Disposition.Name)
	}

	t.Run("finds by GHL id with its relations", func(t *testing.T) {
		found, err := repo.FindCall("call-1")
		require.NoError(t, err)
		assertRelations(t, found)
	})

	t.Run("finds by id with its relations", func(t *testing.T) {
		found, err := repo.FindCallById(call.ID)
		require.NoError(t, err)
		assertRelations(t, found)
	})

	t.Run("finds by SIP session id with its relations", func(t *testing.T) {
		found, err := repo.FindCallBySIPSessionId("sip-1")
		require.NoError(t, err)
		assertRelations(t, found)
	})

	t.Run("unknown ids", func(t *testing.T) {
		_, err := repo.FindCall("unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = repo.FindCallById(999)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = repo.FindCallBySIPSessionId("unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func testFindCalls(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	now := time.Now()
	first := saveCall(t, repo, "call-1", now.Add(-2*time.Hour), nil)
	second := saveCall(t, repo, "call-2", now.Add(-time.Hour), nil)
	saveCall(t, repo, "call-3", now, nil)

	t.Run("calls with the GHL ids, the newest first", func(t *testing.T) {
		calls, err := repo.FindCalls([]string{"call-1", "call-2", "unknown"})
		require.NoError(t, err)
		assert.Equal(t, []string{"call-2", "call-1"}, callGHLIds(calls))
	})

	t.Run("calls with the ids, the newest first", func(t *testing.T) {
		calls, err := repo.FindCallsByIds([]uint{first.ID, second.ID, 999})
		require.NoError(t, err)
		assert.Equal(t, []string{"call-2", "call-1"}, callGHLIds(calls))
	})

	t.Run("no ids", func(t *testing.T) {
		calls, err := repo.FindCalls([]string{})
		require.NoError(t, err)
		assert.Empty(t, calls)
	})

	t.Run("counts all the calls", func(t *testing.T) {
		count, err := repo.CountCalls()
		require.NoError(t, err)
		assert.EqualValues(t, 3, count)
	})
}

func testFindLatestCallHistory(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	user := saveUser(t, repo, "user-1", "User 1")
	other := saveUser(t, repo, "user-2", "User 2")
	now := time.Now()
	for i, ghlId := range []string{"call-1", "call-2", "call-3"} {
		saveCall(t, repo, ghlId, now.Add(time.Duration(i)*time.Minute), func(call *ldspec.DbCall) {
			call.UserId = &user.ID
		})
	}
	saveCall(t, repo, "call-other", now.Add(time.Hour), func(call *ldspec.DbCall) {
		call.UserId = &other.ID
	})

	t.Run("calls of the user, the newest first", func(t *testing.T) {
		calls, err := repo.FindLatestCallHistory(user.ID, 10)
		require.NoError(t, err)
		assert.Equal(t, []string{"call-3", "call-2", "call-1"}, callGHLIds(calls))
	})

	t.Run("up to the limit", func(t *testing.T) {
		calls, err := repo.FindLatestCallHistory(user.ID, 2)
		require.NoError(t, err)
		assert.Equal(t, []string{"call-3", "call-2"}, callGHLIds(calls))
	})

	t.Run("user without calls", func(t *testing.T) {
		calls, err := repo.FindLatestCallHistory(999, 10)
		require.NoError(t, err)
		assert.Empty(t, calls)
	})
}

func testFindCallHistory(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	user := saveUser(t, repo, "user-1", "User 1")
	other := saveUser(t, repo, "user-2", "User 2")
	contact := saveContact(t, repo, "contact-1", "+15550000001")
	now := time.Now().UTC().Truncate(time.Second)
	// call-1 is the oldest; the even calls are outbound and completed
	for i, ghlId := range []string{"call-1", "call-2", "call-3", "call-4"} {
		saveCall(t, repo, ghlId, now.Add(time.Duration(i)*time.Minute), func(call *ldspec.DbCall) {
			call.UserId = &user.ID
			call.GHLUserId = user.GHLId
			call.Outbound = ldutils.Ptr(i%2 == 1)
			if i%2 == 1 {
				call.CallStatus = ldutils.Ptr("cmp")
			}
			if i < 2 {
				call.ContactId = &contact.ID
				call.GHLContactId = &contact.GHLId
			}
		})
	}
	saveCall(t, repo, "call-other", now.Add(time.Hour), func(call *ldspec.DbCall) {
		call.UserId = &other.ID
		call.GHLUserId = other.GHLId
	})
	// imported from GHL before its user: only the GHL id of the user is set
	saveCall(t, repo, "call-not-imported", now.Add(-time.Hour), func(call *ldspec.DbCall) {
		call.GHLUserId = ldutils.Ptr("user-3")
	})

	find := func(t *testing.T, filter ldspec.DbCallHistoryFilter) ([]string, int64) {
		t.Helper()
		calls, total, err := repo.FindCallHistory(&filter)
		require.NoError(t, err)
		return callGHLIds(calls), total
	}

	t.Run("all the calls, the newest first", func(t *testing.T) {
		ids, total := find(t, ldspec.DbCallHistoryFilter{Limit: 10})
		assert.Equal(t, []string{"call-other", "call-4", "call-3", "call-2", "call-1", "call-not-imported"}, ids)
		assert.EqualValues(t, 6, total)
	})

	t.Run("a page with the total of all the pages", func(t *testing.T) {
		ids, total := find(t, ldspec.DbCallHistoryFilter{Limit: 2, Offset: 1})
		assert.Equal(t, []string{"call-4", "call-3"}, ids)
		assert.EqualValues(t, 6, total)
	})

	t.Run("by user GHL id", func(t *testing.T) {
		ids, total := find(t, ldspec.DbCallHistoryFilter{Limit: 10, UserGHLIds: []string{"user-2"}})
		assert.Equal(t, []string{"call-other"}, ids)
		assert.EqualValues(t, 1, total)

		ids, _ = find(t, ldspec.DbCallHistoryFilter{Limit: 10, UserGHLIds: []string{"user-3"}})
		assert.Equal(t, []string{"call-not-imported"}, ids)
	})

	t.Run("by contact GHL id", func(t *testing.T) {
		ids, _ := find(t, ldspec.DbCallHistoryFilter{Limit: 10, ContactGHLIds: []string{"contact-1"}})
		assert.Equal(t, []string{"call-2", "call-1"}, ids)
	})

	t.Run("unknown GHL ids", func(t *testing.T) {
		ids, total := find(t, ldspec.DbCallHistoryFilter{Limit: 10, UserGHLIds: []string{"unknown"}})
		assert.Empty(t, ids)
		assert.Zero(t, total)
	})

	t.Run("by date, direction and status", func(t *testing.T) {
		ids, _ := find(t, ldspec.DbCallHistoryFilter{Limit: 10, From: ldutils.Ptr(now.Add(time.Minute)), To: ldutils.Ptr(now.Add(3 * time.Minute))})
		assert.Equal(t, []string{"call-3", "call-2"}, ids)

		ids, _ = find(t, ldspec.DbCallHistoryFilter{Limit: 10, UserGHLIds: []string{"user-1"}, Outbound: ldutils.Ptr(false)})
		assert.Equal(t, []string{"call-3", "call-1"}, ids)

		ids, _ = find(t, ldspec.DbCallHistoryFilter{Limit: 10, CallStatus: ldutils.Ptr("cmp")})
		assert.Equal(t, []string{"call-4", "call-2"}, ids)
	})
}

func testFindCallsByContact(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	contact := saveContact(t, repo, "contact-1", "+15550000001")
	other := saveContact(t, repo, "contact-2", "+15550000002")
	saveContact(t, repo, "contact-3", "+15550000003")
	now := time.Now()
	saveCall(t, repo, "call-1", now.Add(-time.Hour), func(call *ldspec.DbCall) { call.ContactId = &contact.ID })
	saveCall(t, repo, "call-2", now, func(call *ldspec.DbCall) { call.ContactId = &contact.ID })
	saveCall(t, repo, "call-other", now, func(call *ldspec.DbCall) { call.ContactId = &other.ID })

	t.Run("calls of the contact, the newest first", func(t *testing.T) {
		calls, err := repo.FindCallsByContact("contact-1")
		require.NoError(t, err)
		assert.Equal(t, []string{"call-2", "call-1"}, callGHLIds(calls))
	})

	t.Run("contact without calls", func(t *testing.T) {
		calls, err := repo.FindCallsByContact("contact-3")
		require.NoError(t, err)
		assert.Empty(t, calls)
	})

	t.Run("unknown contact", func(t *testing.T) {
		_, err := repo.FindCallsByContact("unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func testFindCallsByDate(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	contact := saveContact(t, repo, "contact-1", "+15550000001")
	date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	withContact := func(call *ldspec.DbCall) { call.ContactId = &contact.ID }
	saveCall(t, repo, "call-before", date.Add(-time.Minute), withContact)
	saveCall(t, repo, "call-morning", date.Add(9*time.Hour), withContact)
	saveCall(t, repo, "call-evening", date.Add(20*time.Hour), withContact)
	saveCall(t, repo, "call-no-contact", date.Add(10*time.Hour), nil)
	saveCall(t, repo, "call-next-day", date.Add(25*time.Hour), withContact)

	t.Run("calls with a contact in the 24 hours from the date, the newest first", func(t *testing.T) {
		calls, err := repo.FindCallsByDate(date)
		require.NoError(t, err)
		assert.Equal(t, []string{"call-evening", "call-morning"}, callGHLIds(calls))
	})

	t.Run("date without calls", func(t *testing.T) {
		calls, err := repo.FindCallsByDate(date.AddDate(0, 1, 0))
		require.NoError(t, err)
		assert.Empty(t, calls)
	})
}
