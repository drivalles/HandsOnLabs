package ldtenant_test

import (
	"fmt"
	ldconn "ld-backend/internal/impl/back/db/conn"
	ldtenant "ld-backend/internal/impl/back/db/tenant"
	ldutils "ld-backend/internal/impl/core/utils"
	ldspec "ld-backend/internal/spec"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	ldconn.RunTestsWithDbs(m)
}

// TestTenantRepo runs all the repo cases with each database driver.
func TestTenantRepo(t *testing.T) {
	for _, d := range ldconn.TestDrivers() {
		t.Run(d.Name, func(t *testing.T) {
			d.Check(t)

			t.Run("Migrations", func(t *testing.T) {
				testMigrations(t, d)
			})

			t.Run("CacheExpired", func(t *testing.T) {
				testCacheExpired(t, d)
			})

			t.Run("UpdateCache", func(t *testing.T) {
				testUpdateCache(t, d)
			})

			t.Run("DeleteCache", func(t *testing.T) {
				testDeleteCache(t, d)
			})

			t.Run("FindNextExpirationTime", func(t *testing.T) {
				testFindNextExpirationTime(t, d)
			})

			t.Run("SaveCall", func(t *testing.T) {
				testSaveCall(t, d)
			})

			t.Run("DeleteCall", func(t *testing.T) {
				testDeleteCall(t, d)
			})

			t.Run("FindCall", func(t *testing.T) {
				testFindCall(t, d)
			})

			t.Run("FindCalls", func(t *testing.T) {
				testFindCalls(t, d)
			})

			t.Run("FindLatestCallHistory", func(t *testing.T) {
				testFindLatestCallHistory(t, d)
			})

			t.Run("FindCallHistory", func(t *testing.T) {
				testFindCallHistory(t, d)
			})

			t.Run("FindCallsByContact", func(t *testing.T) {
				testFindCallsByContact(t, d)
			})

			t.Run("FindCallsByDate", func(t *testing.T) {
				testFindCallsByDate(t, d)
			})

			t.Run("SaveContact", func(t *testing.T) {
				testSaveContact(t, d)
			})

			t.Run("FindContact", func(t *testing.T) {
				testFindContact(t, d)
			})

			t.Run("FindContactByPhoneNumber", func(t *testing.T) {
				testFindContactByPhoneNumber(t, d)
			})

			t.Run("FindContactTags", func(t *testing.T) {
				testFindContactTags(t, d)
			})

			t.Run("UpdateContactsNoCalls", func(t *testing.T) {
				testUpdateContactsNoCalls(t, d)
			})

			t.Run("UpdateContactCallStats", func(t *testing.T) {
				testUpdateContactCallStats(t, d)
			})

			t.Run("LockContact", func(t *testing.T) {
				testLockContact(t, d)
			})

			t.Run("ContactCallback", func(t *testing.T) {
				testContactCallback(t, d)
			})

			t.Run("CountAbandonedLeads", func(t *testing.T) {
				testCountAbandonedLeads(t, d)
			})

			t.Run("ReleaseAllLocks", func(t *testing.T) {
				testReleaseAllLocks(t, d)
			})

			t.Run("LockMaintenance", func(t *testing.T) {
				testLockMaintenance(t, d)
			})

			t.Run("FindOrCreateDid", func(t *testing.T) {
				testFindOrCreateDid(t, d)
			})

			t.Run("FindDids", func(t *testing.T) {
				testFindDids(t, d)
			})

			t.Run("LockDid", func(t *testing.T) {
				testLockDid(t, d)
			})

			t.Run("UpdateDidCallStats", func(t *testing.T) {
				testUpdateDidCallStats(t, d)
			})

			t.Run("DidHealthReports", func(t *testing.T) {
				testDidHealthReports(t, d)
			})

			t.Run("SaveOpportunity", func(t *testing.T) {
				testSaveOpportunity(t, d)
			})

			t.Run("FindOpportunities", func(t *testing.T) {
				testFindOpportunities(t, d)
			})

			t.Run("FindNewOpportunities", func(t *testing.T) {
				testFindNewOpportunities(t, d)
			})

			t.Run("FindCallbackOpportunities", func(t *testing.T) {
				testFindCallbackOpportunities(t, d)
			})

			t.Run("SavePipeline", func(t *testing.T) {
				testSavePipeline(t, d)
			})

			t.Run("FindPipelines", func(t *testing.T) {
				testFindPipelines(t, d)
			})

			t.Run("SaveStage", func(t *testing.T) {
				testSaveStage(t, d)
			})

			t.Run("FindStages", func(t *testing.T) {
				testFindStages(t, d)
			})

			t.Run("DeleteStagesByPipelineId", func(t *testing.T) {
				testDeleteStagesByPipelineId(t, d)
			})

			t.Run("FindOrCreateUser", func(t *testing.T) {
				testFindOrCreateUser(t, d)
			})

			t.Run("FindUsers", func(t *testing.T) {
				testFindUsers(t, d)
			})

			t.Run("UpdateUserAccess", func(t *testing.T) {
				testUpdateUserAccess(t, d)
			})

			t.Run("NewLeadsStage", func(t *testing.T) {
				testNewLeadsStage(t, d)
			})

			t.Run("DispositionField", func(t *testing.T) {
				testDispositionField(t, d)
			})

			t.Run("SaveDialerSession", func(t *testing.T) {
				testSaveDialerSession(t, d)
			})

			t.Run("FindDialerSessions", func(t *testing.T) {
				testFindDialerSessions(t, d)
			})

			t.Run("RemoveOldDialerSessions", func(t *testing.T) {
				testRemoveOldDialerSessions(t, d)
			})

			t.Run("SaveTransferContact", func(t *testing.T) {
				testSaveTransferContact(t, d)
			})

			t.Run("SaveDisposition", func(t *testing.T) {
				testSaveDisposition(t, d)
			})

			t.Run("FindDispositions", func(t *testing.T) {
				testFindDispositions(t, d)
			})
			t.Run("FindOrCreateDispositionByName", func(t *testing.T) {
				testFindOrCreateDispositionByName(t, d)
			})

			t.Run("SaveCallScript", func(t *testing.T) {
				testSaveCallScript(t, d)
			})

			t.Run("ConcurrentLocks", func(t *testing.T) {
				testConcurrentLocks(t, d)
			})
		})
	}
}

func testMigrations(t *testing.T, d ldconn.TestDriver) {
	t.Run("migrates the tenant tables", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		migrator := repo.Db().Migrator()
		for _, table := range ldspec.DbTenantTables {
			assert.True(t, migrator.HasTable(table), "missing table for %T", table)
		}
	})

	t.Run("records the schema version", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		version, err := ldconn.SchemaVersion(repo.Db())
		require.NoError(t, err)
		assert.Equal(t, ldspec.DB_SCHEMA_VERSION, version)
	})

	t.Run("migrates an existing database again", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		saveUser(t, repo, "user-1", "User 1")
		require.NoError(t, repo.Db().AutoMigrate(ldspec.DbTenantTables...))

		users, err := repo.FindUsers()
		require.NoError(t, err)
		assert.Len(t, users, 1, "the migration keeps the data")
	})
}

// testConcurrentLocks locks the same records from several workers at once;
// only one lock holds each. The workers are few, and the cases are skipped
// when the server limits the open transactions, as the cockroachdb test
// server without a license.
func testConcurrentLocks(t *testing.T, d ldconn.TestDriver) {
	d.SkipConcurrentWrites(t)
	const workers = 4

	t.Run("one lock holds the contact", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		saveContact(t, repo, "contact-1", "+15550000001")

		locked := runConcurrently(t, workers, func(i int) (bool, error) {
			return repo.LockContact("contact-1", fmt.Sprint("node-", i), user.ID)
		})
		assert.Equal(t, 1, locked)
	})

	t.Run("one lock holds the did", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		saveDid(t, repo, "15550000001", true)

		locked := runConcurrently(t, workers, func(i int) (bool, error) {
			return repo.LockDid("15550000001", fmt.Sprint("node-", i), user.ID)
		})
		assert.Equal(t, 1, locked)
	})

	t.Run("one caller renews an expired cache", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		repo.UpdateCache("pipelines", -time.Minute)

		expired := runConcurrently(t, workers, func(int) (bool, error) {
			return repo.CacheExpired("pipelines", time.Hour), nil
		})
		assert.Equal(t, 1, expired)
	})
}

// runConcurrently runs fn from the workers at once, returning how many
// returned true; none must fail.
func runConcurrently(t *testing.T, workers int, fn func(i int) (bool, error)) int {
	t.Helper()
	var wg sync.WaitGroup
	results := make(chan bool, workers)
	errs := make(chan error, workers)
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := fn(i)
			errs <- err
			results <- ok
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	close(results)
	for err := range errs {
		assert.NoError(t, err)
	}
	count := 0
	for ok := range results {
		if ok {
			count++
		}
	}
	return count
}

// saveUser saves a user with the GHL id and name.
func saveUser(t *testing.T, repo ldspec.TenantRepo, ghlId string, name string) *ldspec.DbUser {
	t.Helper()
	user := &ldspec.DbUser{GHLId: ldutils.Ptr(ghlId), UserName: ldutils.Ptr(name)}
	require.NoError(t, repo.SaveUser(user))
	return user
}

// saveContact saves a contact with the GHL id and phone number.
func saveContact(t *testing.T, repo ldspec.TenantRepo, ghlId string, phoneNumber string) *ldspec.DbContact {
	t.Helper()
	contact := &ldspec.DbContact{GHLId: ghlId, PhoneNumber: phoneNumber, FirstName: "First " + ghlId}
	require.NoError(t, repo.SaveContact(contact))
	return contact
}

// saveDid saves a DID with the number.
func saveDid(t *testing.T, repo ldspec.TenantRepo, number string, enabled bool) *ldspec.DbDid {
	t.Helper()
	did := &ldspec.DbDid{DidNumber: number, Enabled: ldutils.Ptr(enabled)}
	require.NoError(t, repo.SaveDid(did))
	return did
}

// saveCall saves a call with the GHL id at the time.
func saveCall(t *testing.T, repo ldspec.TenantRepo, ghlId string, at time.Time, fill func(call *ldspec.DbCall)) *ldspec.DbCall {
	t.Helper()
	call := &ldspec.DbCall{GHLId: ldutils.Ptr(ghlId), PhoneNumber: "+15550000001", DateTime: ldutils.Ptr(at)}
	if fill != nil {
		fill(call)
	}
	require.NoError(t, repo.SaveCall(call))
	return call
}

// savePipeline saves a pipeline with the GHL id.
func savePipeline(t *testing.T, repo ldspec.TenantRepo, ghlId string) *ldspec.DbPipeline {
	t.Helper()
	pipeline := &ldspec.DbPipeline{GHLId: ghlId, Name: "Pipeline " + ghlId}
	require.NoError(t, repo.SavePipeline(pipeline))
	return pipeline
}

// saveStage saves a stage of the pipeline with the GHL id.
func saveStage(t *testing.T, repo ldspec.TenantRepo, pipeline *ldspec.DbPipeline, ghlId string) *ldspec.DbPipelineStage {
	t.Helper()
	stage := &ldspec.DbPipelineStage{PipelineId: pipeline.ID, GHLId: ghlId, Name: "Stage " + ghlId}
	require.NoError(t, repo.SaveStage(stage))
	return stage
}

// callGHLIds are the GHL ids of the calls, in their order.
func callGHLIds(calls []*ldspec.DbCall) []string {
	ids := make([]string, 0, len(calls))
	for _, call := range calls {
		ids = append(ids, *call.GHLId)
	}
	return ids
}
