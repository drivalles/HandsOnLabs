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

func testFindOrCreateDid(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates the did of the validated number", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		did, err := repo.FindOrCreateDid(ldutils.Ptr("+1 (555) 000-0001"), true)
		require.NoError(t, err)
		assert.NotZero(t, did.ID)
		assert.Equal(t, "15550000001", did.DidNumber)
		assert.True(t, *did.Enabled)

		found, err := repo.FindDid("15550000001")
		require.NoError(t, err)
		assert.Equal(t, did.ID, found.ID)
	})

	t.Run("finds the existing did of the number", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		created, err := repo.FindOrCreateDid(ldutils.Ptr("15550000001"), false)
		require.NoError(t, err)

		did, err := repo.FindOrCreateDid(ldutils.Ptr("5550000001"), true)
		require.NoError(t, err)
		assert.Equal(t, created.ID, did.ID)
		assert.False(t, *did.Enabled, "the existing did is kept as is")
		dids, err := repo.FindAllDids()
		require.NoError(t, err)
		assert.Len(t, dids, 1)
	})

	t.Run("a conflicting transaction is tried again", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		creates := ldconn.ConflictOnCreate(t, repo.Db(), "dids", 1)

		did, err := repo.FindOrCreateDid(ldutils.Ptr("15550000001"), true)
		require.NoError(t, err)
		assert.EqualValues(t, 2, creates.Load())
		dids, err := repo.FindAllDids()
		require.NoError(t, err)
		require.Len(t, dids, 1)
		assert.Equal(t, did.ID, dids[0].ID)
	})

	t.Run("nil number", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		did, err := repo.FindOrCreateDid(nil, true)
		assert.Error(t, err)
		assert.Nil(t, did)
	})

	t.Run("invalid number", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		did, err := repo.FindOrCreateDid(ldutils.Ptr("123"), true)
		assert.ErrorContains(t, err, "invalid did number 123")
		assert.Nil(t, did)
		dids, err := repo.FindAllDids()
		require.NoError(t, err)
		assert.Empty(t, dids)
	})
}

func testFindDids(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	saveDid(t, repo, "15550000001", true)
	disabled := saveDid(t, repo, "15550000002", false)
	saveDid(t, repo, "15550000003", true)
	require.NoError(t, repo.SaveDid(&ldspec.DbDid{DidNumber: "15550000004"}))

	didNumbers := func(dids []*ldspec.DbDid) []string {
		numbers := make([]string, 0, len(dids))
		for _, did := range dids {
			numbers = append(numbers, did.DidNumber)
		}
		return numbers
	}

	t.Run("enabled dids", func(t *testing.T) {
		dids, err := repo.FindEnabledDids()
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"15550000001", "15550000003"}, didNumbers(dids))
	})

	t.Run("all the dids", func(t *testing.T) {
		dids, err := repo.FindAllDids()
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"15550000001", "15550000002", "15550000003", "15550000004"}, didNumbers(dids))
	})

	t.Run("finds by number", func(t *testing.T) {
		did, err := repo.FindDid("15550000002")
		require.NoError(t, err)
		assert.Equal(t, disabled.ID, did.ID)
		assert.False(t, *did.Enabled)
	})

	t.Run("unknown number", func(t *testing.T) {
		_, err := repo.FindDid("15559999999")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("updates a did", func(t *testing.T) {
		did, err := repo.FindDid("15550000003")
		require.NoError(t, err)
		did.Enabled = ldutils.Ptr(false)
		did.IsSpam = ldutils.Ptr(true)
		require.NoError(t, repo.SaveDid(did))

		found, err := repo.FindDid("15550000003")
		require.NoError(t, err)
		assert.Equal(t, did.ID, found.ID)
		assert.False(t, *found.Enabled)
		assert.True(t, *found.IsSpam)
	})

	t.Run("duplicated number", func(t *testing.T) {
		assert.Error(t, repo.SaveDid(&ldspec.DbDid{DidNumber: "15550000001"}))
	})
}

func testLockDid(t *testing.T, d ldconn.TestDriver) {
	t.Run("locks a free did", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		saveDid(t, repo, "15550000001", true)

		locked, err := repo.LockDid("15550000001", "node-1", user.ID)
		require.NoError(t, err)
		assert.True(t, locked)

		found, err := repo.FindDid("15550000001")
		require.NoError(t, err)
		assert.Equal(t, "node-1", *found.LockId)
		assert.Equal(t, user.ID, *found.LockUserId)
		require.NotNil(t, found.LockTs)
		assert.WithinDuration(t, time.Now(), *found.LockTs, time.Minute)
	})

	t.Run("the same lock locks it again", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		saveDid(t, repo, "15550000001", true)
		_, err := repo.LockDid("15550000001", "node-1", user.ID)
		require.NoError(t, err)

		locked, err := repo.LockDid("15550000001", "node-1", user.ID)
		require.NoError(t, err)
		assert.True(t, locked)
	})

	t.Run("another lock does not lock it", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		saveDid(t, repo, "15550000001", true)
		_, err := repo.LockDid("15550000001", "node-1", user.ID)
		require.NoError(t, err)

		locked, err := repo.LockDid("15550000001", "node-2", user.ID)
		require.NoError(t, err)
		assert.False(t, locked)
		found, err := repo.FindDid("15550000001")
		require.NoError(t, err)
		assert.Equal(t, "node-1", *found.LockId)
	})

	t.Run("a lock older than 10 minutes is taken by another lock", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		other := saveUser(t, repo, "user-2", "User 2")
		require.NoError(t, repo.SaveDid(&ldspec.DbDid{
			DidNumber:  "15550000001",
			LockId:     ldutils.Ptr("node-1"),
			LockTs:     ldutils.Ptr(time.Now().Add(-11 * time.Minute)),
			LockUserId: &user.ID,
		}))

		locked, err := repo.LockDid("15550000001", "node-2", other.ID)
		require.NoError(t, err)
		assert.True(t, locked)
		found, err := repo.FindDid("15550000001")
		require.NoError(t, err)
		assert.Equal(t, "node-2", *found.LockId)
		assert.Equal(t, other.ID, *found.LockUserId)
	})

	t.Run("unknown did", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		locked, err := repo.LockDid("15559999999", "node-1", 1)
		require.NoError(t, err)
		assert.False(t, locked)
	})

	t.Run("the lock unlocks it", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		saveDid(t, repo, "15550000001", true)
		_, err := repo.LockDid("15550000001", "node-1", user.ID)
		require.NoError(t, err)

		unlocked, err := repo.UnlockDid("15550000001", "node-1")
		require.NoError(t, err)
		assert.True(t, unlocked)
		found, err := repo.FindDid("15550000001")
		require.NoError(t, err)
		assert.Nil(t, found.LockId)
		assert.Nil(t, found.LockTs)
		assert.Nil(t, found.LockUserId)
	})

	t.Run("another lock does not unlock it", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		saveDid(t, repo, "15550000001", true)
		_, err := repo.LockDid("15550000001", "node-1", user.ID)
		require.NoError(t, err)

		unlocked, err := repo.UnlockDid("15550000001", "node-2")
		require.NoError(t, err)
		assert.False(t, unlocked)
	})
}

func testUpdateDidCallStats(t *testing.T, d ldconn.TestDriver) {
	t.Run("counts the calls of the last 10 days and sets the latest", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		saveDid(t, repo, "15550000001", true)
		now := time.Now()
		ofDid := func(call *ldspec.DbCall) { call.DidNumber = ldutils.Ptr("15550000001") }
		saveCall(t, repo, "call-old", now.Add(-11*24*time.Hour), ofDid)
		saveCall(t, repo, "call-1", now.Add(-time.Hour), ofDid)
		latest := saveCall(t, repo, "call-2", now, ofDid)
		saveCall(t, repo, "call-other", now.Add(time.Hour), func(call *ldspec.DbCall) {
			call.DidNumber = ldutils.Ptr("15550000002")
		})

		repo.UpdateDidCallStats("15550000001")

		found, err := repo.FindDid("15550000001")
		require.NoError(t, err)
		assert.EqualValues(t, 2, found.NumberOfCalls)
		require.NotNil(t, found.LastCallId)
		assert.Equal(t, latest.ID, *found.LastCallId)
	})

	t.Run("a did without calls has none", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		did := saveDid(t, repo, "15550000001", true)
		did.NumberOfCalls = 5
		require.NoError(t, repo.SaveDid(did))

		repo.UpdateDidCallStats("15550000001")

		found, err := repo.FindDid("15550000001")
		require.NoError(t, err)
		assert.EqualValues(t, 0, found.NumberOfCalls)
		assert.Nil(t, found.LastCallId)
	})

	t.Run("empty and unknown numbers are ignored", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		repo.UpdateDidCallStats("")
		repo.UpdateDidCallStats("15559999999")
		dids, err := repo.FindAllDids()
		require.NoError(t, err)
		assert.Empty(t, dids, "no did is created")
	})
}

func testDidHealthReports(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	did := saveDid(t, repo, "15550000001", true)
	other := saveDid(t, repo, "15550000002", true)
	first := &ldspec.DbDidHealthReport{DidId: &did.ID, IsSpam: ldutils.Ptr(false), SpamScore: ldutils.Ptr(int16(10))}
	require.NoError(t, repo.SaveDidHealthReport(first))
	second := &ldspec.DbDidHealthReport{DidId: &did.ID, IsSpam: ldutils.Ptr(true), SpamScore: ldutils.Ptr(int16(90)), Carrier: ldutils.Ptr("carrier")}
	require.NoError(t, repo.SaveDidHealthReport(second))
	require.NoError(t, repo.SaveDidHealthReport(&ldspec.DbDidHealthReport{DidId: &other.ID}))

	t.Run("reports of the did, the newest first", func(t *testing.T) {
		reports, err := repo.FindDidHealthReports(did.ID)
		require.NoError(t, err)
		require.Len(t, reports, 2)
		assert.Equal(t, second.ID, reports[0].ID)
		assert.Equal(t, first.ID, reports[1].ID)
		assert.True(t, *reports[0].IsSpam)
		assert.EqualValues(t, 90, *reports[0].SpamScore)
		assert.Equal(t, "carrier", *reports[0].Carrier)
	})

	t.Run("did without reports", func(t *testing.T) {
		reports, err := repo.FindDidHealthReports(999)
		require.NoError(t, err)
		assert.Empty(t, reports)
	})

	t.Run("updates a report", func(t *testing.T) {
		first.SpamScore = ldutils.Ptr(int16(20))
		require.NoError(t, repo.SaveDidHealthReport(first))
		reports, err := repo.FindDidHealthReports(did.ID)
		require.NoError(t, err)
		require.Len(t, reports, 2)
		assert.EqualValues(t, 20, *reports[1].SpamScore)
	})
}
