package ldtenant_test

import (
	"fmt"
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

func testSaveContact(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates and updates a contact", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		contact := saveContact(t, repo, "contact-1", "+15550000001")
		require.NotZero(t, contact.ID)

		contact.LastName = "Last"
		require.NoError(t, repo.SaveContact(contact))

		found, err := repo.FindContactByGHLId("contact-1")
		require.NoError(t, err)
		assert.Equal(t, contact.ID, found.ID)
		assert.Equal(t, "Last", found.LastName)
		assert.Equal(t, "+15550000001", found.PhoneNumber)
	})

	t.Run("duplicated GHL id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		saveContact(t, repo, "contact-1", "+15550000001")
		assert.Error(t, repo.SaveContact(&ldspec.DbContact{GHLId: "contact-1"}))
	})
}

func testFindContact(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	saveContact(t, repo, "contact-1", "+15550000001")
	contact := saveContact(t, repo, "contact-2", "+15550000002")

	t.Run("finds by GHL id", func(t *testing.T) {
		found, err := repo.FindContactByGHLId("contact-2")
		require.NoError(t, err)
		assert.Equal(t, contact.ID, found.ID)
	})

	t.Run("finds by id", func(t *testing.T) {
		found, err := repo.FindContactById(contact.ID)
		require.NoError(t, err)
		assert.Equal(t, "contact-2", found.GHLId)
	})

	t.Run("unknown ids", func(t *testing.T) {
		_, err := repo.FindContactByGHLId("unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = repo.FindContactById(999)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func testFindContactByPhoneNumber(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	plus := saveContact(t, repo, "contact-plus", "+15550000001")
	withOne := saveContact(t, repo, "contact-one", "15550000002")
	bare := saveContact(t, repo, "contact-bare", "5550000003")

	t.Run("matches the US formats", func(t *testing.T) {
		for number, want := range map[string]uint{
			"+15550000001":    plus.ID,
			"5550000001":      plus.ID,
			"(555) 000-0002":  withOne.ID,
			"+1 555 000 0003": bare.ID,
		} {
			found, err := repo.FindContactByPhoneNumber(number)
			require.NoError(t, err, number)
			assert.Equal(t, want, found.ID, number)
		}
	})

	t.Run("the newest contact with the number", func(t *testing.T) {
		saveContact(t, repo, "contact-old", "+15550000004")
		newest := saveContact(t, repo, "contact-new", "5550000004")

		found, err := repo.FindContactByPhoneNumber("+15550000004")
		require.NoError(t, err)
		assert.Equal(t, newest.ID, found.ID)
	})

	t.Run("unknown number", func(t *testing.T) {
		_, err := repo.FindContactByPhoneNumber("+15559999999")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("invalid number", func(t *testing.T) {
		_, err := repo.FindContactByPhoneNumber("+44 20 7946 0000")
		assert.Error(t, err)
	})
}

func testFindContactTags(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	contact := saveContact(t, repo, "contact-1", "+15550000001")
	other := saveContact(t, repo, "contact-2", "+15550000002")
	untagged := saveContact(t, repo, "contact-3", "+15550000003")
	for _, tag := range []*ldspec.DbContactTag{
		{ContactId: contact.ID, Tags: "hot"},
		{ContactId: contact.ID, Tags: "callback"},
		{ContactId: other.ID, Tags: "cold"},
	} {
		require.NoError(t, repo.Db().Create(tag).Error)
	}

	t.Run("tags of the contact", func(t *testing.T) {
		assert.ElementsMatch(t, []string{"hot", "callback"}, repo.FindContactTags(contact.ID))
	})

	t.Run("contact without tags", func(t *testing.T) {
		tags := repo.FindContactTags(untagged.ID)
		assert.NotNil(t, tags)
		assert.Empty(t, tags)
	})
}

func testUpdateContactsNoCalls(t *testing.T, d ldconn.TestDriver) {
	t.Run("recounts the calls of every contact", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		called := saveContact(t, repo, "contact-1", "+15550000001")
		stale := saveContact(t, repo, "contact-2", "+15550000002")
		stale.NumberOfCalls = 7
		require.NoError(t, repo.SaveContact(stale))
		for _, ghlId := range []string{"call-1", "call-2"} {
			saveCall(t, repo, ghlId, time.Now(), func(call *ldspec.DbCall) { call.ContactId = &called.ID })
		}

		repo.UpdateContactsNoCalls()

		found, err := repo.FindContactById(called.ID)
		require.NoError(t, err)
		assert.EqualValues(t, 2, found.NumberOfCalls)
		found, err = repo.FindContactById(stale.ID)
		require.NoError(t, err)
		assert.EqualValues(t, 0, found.NumberOfCalls, "a contact without calls has none")
	})

	t.Run("no contacts", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		repo.UpdateContactsNoCalls()
	})
}

func testUpdateContactCallStats(t *testing.T, d ldconn.TestDriver) {
	t.Run("counts the calls and sets the latest", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		contact := saveContact(t, repo, "contact-1", "+15550000001")
		now := time.Now().Truncate(time.Second)
		ofContact := func(call *ldspec.DbCall) { call.GHLContactId = ldutils.Ptr("contact-1") }
		saveCall(t, repo, "call-1", now.Add(-time.Hour), ofContact)
		latest := saveCall(t, repo, "call-2", now, ofContact)
		saveCall(t, repo, "call-0", now.Add(-2*time.Hour), ofContact)
		saveCall(t, repo, "call-other", now.Add(time.Hour), func(call *ldspec.DbCall) {
			call.GHLContactId = ldutils.Ptr("contact-2")
		})

		repo.UpdateContactCallStats(ldutils.Ptr("contact-1"))

		found, err := repo.FindContactById(contact.ID)
		require.NoError(t, err)
		assert.EqualValues(t, 3, found.NumberOfCalls)
		require.NotNil(t, found.LastCallId)
		assert.Equal(t, latest.ID, *found.LastCallId)
		require.NotNil(t, found.LastCallTime)
		assert.WithinDuration(t, now, *found.LastCallTime, time.Second)
	})

	t.Run("a contact without calls keeps no latest call", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		contact := saveContact(t, repo, "contact-1", "+15550000001")

		repo.UpdateContactCallStats(ldutils.Ptr("contact-1"))

		found, err := repo.FindContactById(contact.ID)
		require.NoError(t, err)
		assert.EqualValues(t, 0, found.NumberOfCalls)
		assert.Nil(t, found.LastCallId)
	})

	t.Run("nil, empty and unknown ids are ignored", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		repo.UpdateContactCallStats(nil)
		repo.UpdateContactCallStats(ldutils.Ptr(""))
		repo.UpdateContactCallStats(ldutils.Ptr("unknown"))
		var count int64
		require.NoError(t, repo.Db().Model(&ldspec.DbContact{}).Count(&count).Error)
		assert.Zero(t, count, "no contact is created")
	})
}

func testLockContact(t *testing.T, d ldconn.TestDriver) {
	t.Run("locks a free contact", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		contact := saveContact(t, repo, "contact-1", "+15550000001")

		locked, err := repo.LockContact("contact-1", "node-1", user.ID)
		require.NoError(t, err)
		assert.True(t, locked)

		found, err := repo.FindContactById(contact.ID)
		require.NoError(t, err)
		assert.Equal(t, "node-1", *found.LockId)
		assert.Equal(t, user.ID, *found.LockUserId)
		require.NotNil(t, found.LockTs)
		assert.WithinDuration(t, time.Now(), *found.LockTs, time.Minute)
		require.NotNil(t, found.LastCallAttemptTime)
		assert.WithinDuration(t, time.Now(), *found.LastCallAttemptTime, time.Minute)
	})

	t.Run("the same lock locks it again", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		saveContact(t, repo, "contact-1", "+15550000001")
		_, err := repo.LockContact("contact-1", "node-1", user.ID)
		require.NoError(t, err)

		locked, err := repo.LockContact("contact-1", "node-1", user.ID)
		require.NoError(t, err)
		assert.True(t, locked)
	})

	t.Run("another lock does not lock it", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		contact := saveContact(t, repo, "contact-1", "+15550000001")
		_, err := repo.LockContact("contact-1", "node-1", user.ID)
		require.NoError(t, err)

		locked, err := repo.LockContact("contact-1", "node-2", user.ID)
		require.NoError(t, err)
		assert.False(t, locked)
		found, err := repo.FindContactById(contact.ID)
		require.NoError(t, err)
		assert.Equal(t, "node-1", *found.LockId)
	})

	t.Run("unknown contact", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		locked, err := repo.LockContact("unknown", "node-1", 1)
		require.NoError(t, err)
		assert.False(t, locked)
	})

	t.Run("the lock unlocks it", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		contact := saveContact(t, repo, "contact-1", "+15550000001")
		_, err := repo.LockContact("contact-1", "node-1", user.ID)
		require.NoError(t, err)

		unlocked, err := repo.UnlockContact("contact-1", "node-1")
		require.NoError(t, err)
		assert.True(t, unlocked)
		found, err := repo.FindContactById(contact.ID)
		require.NoError(t, err)
		assert.Nil(t, found.LockId)
		assert.Nil(t, found.LockTs)
		assert.Nil(t, found.LockUserId)

		locked, err := repo.LockContact("contact-1", "node-2", user.ID)
		require.NoError(t, err)
		assert.True(t, locked, "another lock can lock it")
	})

	t.Run("another lock does not unlock it", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		saveContact(t, repo, "contact-1", "+15550000001")
		_, err := repo.LockContact("contact-1", "node-1", user.ID)
		require.NoError(t, err)

		unlocked, err := repo.UnlockContact("contact-1", "node-2")
		require.NoError(t, err)
		assert.False(t, unlocked)
	})

	t.Run("a free contact is not unlocked", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		saveContact(t, repo, "contact-1", "+15550000001")
		unlocked, err := repo.UnlockContact("contact-1", "node-1")
		require.NoError(t, err)
		assert.False(t, unlocked)
	})
}

func testContactCallback(t *testing.T, d ldconn.TestDriver) {
	t.Run("schedules the callback of the contact", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		contact := saveContact(t, repo, "contact-1", "+15550000001")
		at := time.Now().Add(time.Hour).Truncate(time.Second)

		require.NoError(t, repo.UpdateContactCallback("contact-1", user.ID, at))

		found, err := repo.FindContactById(contact.ID)
		require.NoError(t, err)
		require.NotNil(t, found.CallbackTime)
		assert.WithinDuration(t, at, *found.CallbackTime, time.Second)
		assert.Equal(t, user.ID, *found.CallbackUserId)
	})

	t.Run("unknown contact", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		err := repo.UpdateContactCallback("unknown", 1, time.Now())
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("clears the callback", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		contact := saveContact(t, repo, "contact-1", "+15550000001")
		require.NoError(t, repo.UpdateContactCallback("contact-1", user.ID, time.Now()))

		require.NoError(t, repo.ClearContactCallback("contact-1"))

		found, err := repo.FindContactById(contact.ID)
		require.NoError(t, err)
		assert.Nil(t, found.CallbackTime)
		assert.Nil(t, found.CallbackUserId)
	})

	t.Run("clearing a contact without a callback or unknown is ignored", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		saveContact(t, repo, "contact-1", "+15550000001")
		assert.NoError(t, repo.ClearContactCallback("contact-1"))
		assert.NoError(t, repo.ClearContactCallback("unknown"))
	})
}

func testCountAbandonedLeads(t *testing.T, d ldconn.TestDriver) {
	t.Run("contacts registered more than an hour ago and never called", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		old := time.Now().Add(-2 * time.Hour)
		recent := time.Now().Add(-10 * time.Minute)
		for _, contact := range []*ldspec.DbContact{
			{GHLId: "abandoned-1", DateRegistered: &old},
			{GHLId: "abandoned-2", DateRegistered: &old},
			{GHLId: "called", DateRegistered: &old, NumberOfCalls: 1},
			{GHLId: "recent", DateRegistered: &recent},
			{GHLId: "no-date"},
		} {
			require.NoError(t, repo.SaveContact(contact))
		}

		count, err := repo.CountAbandonedLeads()
		require.NoError(t, err)
		assert.EqualValues(t, 2, count)
	})

	t.Run("no contacts", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		count, err := repo.CountAbandonedLeads()
		require.NoError(t, err)
		assert.Zero(t, count)
	})
}

func testReleaseAllLocks(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	user := saveUser(t, repo, "user-1", "User 1")
	saveContact(t, repo, "contact-1", "+15550000001")
	saveContact(t, repo, "contact-2", "+15550000002")
	saveDid(t, repo, "15550000011", true)
	saveDid(t, repo, "15550000012", true)
	for _, lock := range []struct{ contact, did, node string }{
		{"contact-1", "15550000011", "node-1"},
		{"contact-2", "15550000012", "node-2"},
	} {
		locked, err := repo.LockContact(lock.contact, lock.node, user.ID)
		require.NoError(t, err)
		require.True(t, locked)
		locked, err = repo.LockDid(lock.did, lock.node, user.ID)
		require.NoError(t, err)
		require.True(t, locked)
	}

	repo.ReleaseAllLocks("node-1")

	t.Run("releases the contacts and dids of the node", func(t *testing.T) {
		contact, err := repo.FindContactByGHLId("contact-1")
		require.NoError(t, err)
		assert.Nil(t, contact.LockId)
		assert.Nil(t, contact.LockTs)
		assert.Nil(t, contact.LockUserId)
		did, err := repo.FindDid("15550000011")
		require.NoError(t, err)
		assert.Nil(t, did.LockId)
		assert.Nil(t, did.LockTs)
		assert.Nil(t, did.LockUserId)
	})

	t.Run("keeps the locks of the other nodes", func(t *testing.T) {
		contact, err := repo.FindContactByGHLId("contact-2")
		require.NoError(t, err)
		assert.Equal(t, "node-2", *contact.LockId)
		did, err := repo.FindDid("15550000012")
		require.NoError(t, err)
		assert.Equal(t, "node-2", *did.LockId)
	})
}

func testLockMaintenance(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	user := saveUser(t, repo, "user-1", "User 1")
	completed := saveCall(t, repo, "call-completed", time.Now(), func(call *ldspec.DbCall) {
		call.CallStatus = ldutils.Ptr("completed")
	})
	inProgress := saveCall(t, repo, "call-in-progress", time.Now(), func(call *ldspec.DbCall) {
		call.CallStatus = ldutils.Ptr("ringing")
	})
	noStatus := saveCall(t, repo, "call-no-status", time.Now(), nil)
	old := time.Now().Add(-3 * time.Hour)
	recent := time.Now().Add(-time.Hour)
	locks := []struct {
		name     string
		lockTs   time.Time
		lastCall *ldspec.DbCall
		released bool
	}{
		{"old-without-call", old, nil, true},
		{"old-completed", old, completed, true},
		{"old-in-progress", old, inProgress, false},
		{"old-no-status", old, noStatus, false},
		{"recent", recent, nil, false},
	}
	for i, lock := range locks {
		contact := &ldspec.DbContact{GHLId: lock.name, LockId: ldutils.Ptr("node-1"), LockTs: ldutils.Ptr(lock.lockTs), LockUserId: &user.ID}
		did := &ldspec.DbDid{DidNumber: fmt.Sprint("1555000002", i), LockId: ldutils.Ptr("node-1"), LockTs: ldutils.Ptr(lock.lockTs), LockUserId: &user.ID}
		if lock.lastCall != nil {
			contact.LastCallId = &lock.lastCall.ID
			did.LastCallId = &lock.lastCall.ID
		}
		require.NoError(t, repo.SaveContact(contact))
		require.NoError(t, repo.SaveDid(did))
	}

	repo.LockMaintenance("node-1")

	for i, lock := range locks {
		t.Run(lock.name, func(t *testing.T) {
			contact, err := repo.FindContactByGHLId(lock.name)
			require.NoError(t, err)
			did, err := repo.FindDid(fmt.Sprint("1555000002", i))
			require.NoError(t, err)
			if lock.released {
				assert.Nil(t, contact.LockId, "the contact lock is released")
				assert.Nil(t, did.LockId, "the did lock is released")
			} else {
				assert.NotNil(t, contact.LockId, "the contact lock is kept")
				assert.NotNil(t, did.LockId, "the did lock is kept")
			}
		})
	}
}
