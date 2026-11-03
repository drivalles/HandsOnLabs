package core

import (
	"fmt"
	"ld-backend/internal/impl/utils"
	"ld-backend/internal/spec"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	RunTestsWithDbs(m)
}

func testMigrations(t *testing.T, d TestDriver) {
	t.Run("migrates the core tables", func(t *testing.T) {
		repo := newTestRepo(t, d)
		migrator := repo.coreDb.Migrator()
		for _, table := range spec.DbCoreTables {
			assert.True(t, migrator.HasTable(table))
		}
	})

	t.Run("migrates an existing database again", func(t *testing.T) {
		repo := newTestRepo(t, d)
		require.NoError(t, repo.coreDb.AutoMigrate(spec.DbCoreTables...))
	})
}

// coreRepoCases run against every driver in TestCoreRepo.
var coreRepoCases = []struct {
	name string
	run  func(t *testing.T, d TestDriver)
}{
	{"Migrations", testMigrations},
	{"FindTenant", testFindTenant},
	{"SaveTenant", testSaveTenant},
	{"FindCompanies", testFindCompanies},
	{"GetLatestInstall", testGetLatestInstall},
	{"GetLatestAccess", testGetLatestAccess},
	{"FindByCompanyAndAuthCode", testFindByCompanyAndAuthCode},
	{"FindByAuthCode", testFindByAuthCode},
	{"SaveAppInstall", testSaveAppInstall},
	{"FindAllAppInstalls", testFindAllAppInstalls},
	{"CleanAppInstalls", testCleanAppInstalls},
	{"FindInstalledLocations", testFindInstalledLocations},
	{"FindInstalledLocation", testFindInstalledLocation},
	{"SaveInstalledLocation", testSaveInstalledLocation},
	{"FindServiceAccount", testFindServiceAccount},
	{"SaveServiceAccount", testSaveServiceAccount},
	{"FindNode", testFindNode},
	{"FindAllNodes", testFindAllNodes},
	{"SaveNode", testSaveNode},
	{"ConcurrentSaves", testConcurrentSaves},
}

// TestCoreRepo runs all the repo cases with each database driver.
func TestCoreRepo(t *testing.T) {
	for _, d := range TestDrivers() {
		t.Run(d.Name, func(t *testing.T) {
			d.Check(t)
			for _, c := range coreRepoCases {
				t.Run(c.name, func(t *testing.T) {
					c.run(t, d)
				})
			}
		})
	}
}

func TestNewCoreRepo(t *testing.T) {
	t.Run("unsupported driver", func(t *testing.T) {
		dbCfg := spec.DBConfig{
			Driver:     "mysql",
			CoreDBName: "",
			DbUrlTpl:   "some-url",
		}
		repo, err := NewCoreRepo(dbCfg)
		require.Error(t, err)
		assert.Nil(t, repo)
	})
}

func testFindTenant(t *testing.T, d TestDriver) {
	repo := newTestRepo(t, d)
	require.NoError(t, repo.SaveTenant(&spec.DbTenant{BusinessId: "loc-1", DbName: "loc-1", Title: "Location 1"}))
	require.NoError(t, repo.SaveTenant(&spec.DbTenant{BusinessId: "loc-2", DbName: "loc-2", Title: "Location 2"}))

	t.Run("finds by business id", func(t *testing.T) {
		tenant, err := repo.FindTenant("loc-2")
		require.NoError(t, err)
		assert.Equal(t, "loc-2", tenant.BusinessId)
		assert.Equal(t, "Location 2", tenant.Title)
	})

	t.Run("unknown business id", func(t *testing.T) {
		tenant, err := repo.FindTenant("loc-unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, tenant)
	})
}

func testSaveTenant(t *testing.T, d TestDriver) {
	t.Run("creates with the default status", func(t *testing.T) {
		repo := newTestRepo(t, d)
		tenant := &spec.DbTenant{
			BusinessId: "loc-1",
			DbName:     "loc-1",
			Title:      "Location 1",
			Url:        "file:loc-1",
			Driver:     string(spec.SqliteDriver),
		}
		require.NoError(t, repo.SaveTenant(tenant))
		assert.NotZero(t, tenant.ID)

		found, err := repo.FindTenant("loc-1")
		require.NoError(t, err)
		assert.Equal(t, tenant.ID, found.ID)
		assert.Equal(t, spec.LDActive, found.Status)
		assert.Equal(t, "file:loc-1", found.Url)
		assert.Equal(t, string(spec.SqliteDriver), found.Driver)
	})

	t.Run("updates an existing tenant", func(t *testing.T) {
		repo := newTestRepo(t, d)
		tenant := &spec.DbTenant{BusinessId: "loc-1", Title: "Old", Status: spec.LDActive}
		require.NoError(t, repo.SaveTenant(tenant))

		tenant.Title = "New"
		tenant.Status = spec.LDInactive
		require.NoError(t, repo.SaveTenant(tenant))

		found, err := repo.FindTenant("loc-1")
		require.NoError(t, err)
		assert.Equal(t, tenant.ID, found.ID)
		assert.Equal(t, "New", found.Title)
		assert.Equal(t, spec.LDInactive, found.Status)

		var count int64
		require.NoError(t, repo.coreDb.Model(&spec.DbTenant{}).Count(&count).Error)
		assert.EqualValues(t, 1, count)
	})
}

func testFindCompanies(t *testing.T, d TestDriver) {
	t.Run("no installations", func(t *testing.T) {
		repo := newTestRepo(t, d)
		companies, err := repo.FindCompanies()
		require.NoError(t, err)
		assert.NotNil(t, companies)
		assert.Empty(t, companies)
	})

	t.Run("distinct companies without nil ids", func(t *testing.T) {
		repo := newTestRepo(t, d)
		now := time.Now()
		SaveInstalls(t, repo,
			CompanyGrant("company-1", "code-1", now),
			CompanyGrant("company-1", "code-2", now),
			LocationAccess("company-1", "loc-1", "code-3", now),
			CompanyGrant("company-2", "code-4", now),
			&spec.DbGHLAppInstallation{AuthorizationCode: "code-no-company"},
		)

		companies, err := repo.FindCompanies()
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"company-1", "company-2"}, companies)
	})
}

func testGetLatestInstall(t *testing.T, d TestDriver) {
	repo := newTestRepo(t, d)
	now := time.Now()
	SaveInstalls(t, repo,
		CompanyGrant("company-1", "old", now.Add(-2*time.Hour)),
		CompanyGrant("company-1", "latest", now.Add(-1*time.Hour)),
		LocationAccess("company-1", "loc-1", "location-newer", now),
		CompanyGrant("company-2", "other-company", now),
	)

	t.Run("newest company grant", func(t *testing.T) {
		install, err := repo.GetLatestInstall("company-1")
		require.NoError(t, err)
		assert.Equal(t, "latest", install.AuthorizationCode)
		assert.Nil(t, install.LocationId)
	})

	t.Run("ignores location tokens", func(t *testing.T) {
		repo := newTestRepo(t, d)
		SaveInstalls(t, repo, LocationAccess("company-1", "loc-1", "code", now))
		install, err := repo.GetLatestInstall("company-1")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, install)
	})

	t.Run("unknown company", func(t *testing.T) {
		install, err := repo.GetLatestInstall("company-unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, install)
	})
}

func testGetLatestAccess(t *testing.T, d TestDriver) {
	repo := newTestRepo(t, d)
	now := time.Now()
	SaveInstalls(t, repo,
		LocationAccess("company-1", "loc-1", "old", now.Add(-2*time.Hour)),
		LocationAccess("company-1", "loc-1", "latest", now.Add(-1*time.Hour)),
		LocationAccess("company-1", "loc-2", "other-location", now),
		CompanyGrant("company-1", "company", now),
	)

	t.Run("newest location token", func(t *testing.T) {
		install, err := repo.GetLatestAccess("loc-1")
		require.NoError(t, err)
		assert.Equal(t, "latest", install.AuthorizationCode)
		assert.Equal(t, "loc-1", *install.LocationId)
	})

	t.Run("other location", func(t *testing.T) {
		install, err := repo.GetLatestAccess("loc-2")
		require.NoError(t, err)
		assert.Equal(t, "other-location", install.AuthorizationCode)
	})

	t.Run("unknown location", func(t *testing.T) {
		install, err := repo.GetLatestAccess("loc-unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, install)
	})
}

func testFindByCompanyAndAuthCode(t *testing.T, d TestDriver) {
	repo := newTestRepo(t, d)
	now := time.Now()
	older := CompanyGrant("company-1", "code-1", now.Add(-2*time.Hour))
	newer := CompanyGrant("company-1", "code-1", now.Add(-1*time.Hour))
	SaveInstalls(t, repo,
		older,
		newer,
		CompanyGrant("company-1", "code-2", now),
		CompanyGrant("company-2", "code-3", now),
		LocationAccess("company-1", "loc-1", "code-loc", now),
	)

	t.Run("newest grant with the code", func(t *testing.T) {
		install, err := repo.FindByCompanyAndAuthCode("company-1", "code-1")
		require.NoError(t, err)
		assert.Equal(t, newer.ID, install.ID)
	})

	t.Run("code of another company", func(t *testing.T) {
		install, err := repo.FindByCompanyAndAuthCode("company-1", "code-3")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, install)
	})

	t.Run("code of a location token", func(t *testing.T) {
		install, err := repo.FindByCompanyAndAuthCode("company-1", "code-loc")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, install)
	})

	t.Run("unknown code", func(t *testing.T) {
		install, err := repo.FindByCompanyAndAuthCode("company-1", "code-unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, install)
	})
}

func testFindByAuthCode(t *testing.T, d TestDriver) {
	repo := newTestRepo(t, d)
	now := time.Now()
	older := CompanyGrant("company-1", "code-1", now.Add(-2*time.Hour))
	newer := CompanyGrant("company-2", "code-1", now.Add(-1*time.Hour))
	SaveInstalls(t, repo,
		older,
		newer,
		CompanyGrant("company-1", "code-2", now),
		LocationAccess("company-1", "loc-1", "code-loc", now),
	)

	t.Run("newest grant with the code of any company", func(t *testing.T) {
		install, err := repo.FindByAuthCode("code-1")
		require.NoError(t, err)
		assert.Equal(t, newer.ID, install.ID)
		assert.Equal(t, "company-2", *install.CompanyId)
	})

	t.Run("code of a location token", func(t *testing.T) {
		install, err := repo.FindByAuthCode("code-loc")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, install)
	})

	t.Run("unknown code", func(t *testing.T) {
		install, err := repo.FindByAuthCode("code-unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, install)
	})
}

func testSaveAppInstall(t *testing.T, d TestDriver) {
	t.Run("creates with all the fields", func(t *testing.T) {
		repo := newTestRepo(t, d)
		install := &spec.DbGHLAppInstallation{
			AuthorizationCode:  "authorization_code",
			AccessToken:        utils.PtrString("access_token"),
			ApprovedLocations:  utils.PtrString("loc-1,loc-2"),
			CompanyId:          utils.PtrString("company1"),
			ExpiresIn:          utils.PtrFloat32(3600),
			IsBulkInstallation: utils.PtrBool(true),
			LocationId:         utils.PtrString("location1"),
			PlanId:             utils.PtrString("plan1"),
			RefreshToken:       utils.PtrString("refresh_token"),
			Scope:              utils.PtrString("contacts.readonly"),
			TokenType:          utils.PtrString("Bearer"),
			UserId:             utils.PtrString("user1"),
			UserType:           utils.PtrString("Location"),
		}
		require.NoError(t, repo.SaveAppInstall(install))
		assert.NotZero(t, install.ID)

		found, err := repo.GetLatestAccess("location1")
		require.NoError(t, err)
		assert.Equal(t, install.ID, found.ID)
		assert.Equal(t, "authorization_code", found.AuthorizationCode)
		assert.Equal(t, "access_token", *found.AccessToken)
		assert.Equal(t, "loc-1,loc-2", *found.ApprovedLocations)
		assert.Equal(t, "company1", *found.CompanyId)
		assert.InDelta(t, 3600, *found.ExpiresIn, 0.001)
		assert.True(t, *found.IsBulkInstallation)
		assert.Equal(t, "plan1", *found.PlanId)
		assert.Equal(t, "refresh_token", *found.RefreshToken)
		assert.Equal(t, "contacts.readonly", *found.Scope)
		assert.Equal(t, "Bearer", *found.TokenType)
		assert.Equal(t, "user1", *found.UserId)
		assert.Equal(t, "Location", *found.UserType)
	})

	t.Run("updates an existing installation", func(t *testing.T) {
		repo := newTestRepo(t, d)
		install := CompanyGrant("company-1", "code-1", time.Now())
		require.NoError(t, repo.SaveAppInstall(install))

		install.AccessToken = utils.PtrString("refreshed")
		install.RefreshToken = utils.PtrString("new-refresh")
		require.NoError(t, repo.SaveAppInstall(install))

		all, err := repo.FindAllAppInstalls()
		require.NoError(t, err)
		require.Len(t, all, 1)
		assert.Equal(t, install.ID, all[0].ID)
		assert.Equal(t, "refreshed", *all[0].AccessToken)
		assert.Equal(t, "new-refresh", *all[0].RefreshToken)
	})
}

func testFindAllAppInstalls(t *testing.T, d TestDriver) {
	t.Run("no installations", func(t *testing.T) {
		repo := newTestRepo(t, d)
		all, err := repo.FindAllAppInstalls()
		require.NoError(t, err)
		assert.Empty(t, all)
	})

	t.Run("company grants and location tokens", func(t *testing.T) {
		repo := newTestRepo(t, d)
		now := time.Now()
		SaveInstalls(t, repo,
			CompanyGrant("company-1", "code-1", now),
			LocationAccess("company-1", "loc-1", "code-2", now),
			CompanyGrant("company-2", "code-3", now),
		)

		all, err := repo.FindAllAppInstalls()
		require.NoError(t, err)
		codes := make([]string, 0, len(all))
		for _, install := range all {
			codes = append(codes, install.AuthorizationCode)
		}
		assert.ElementsMatch(t, []string{"code-1", "code-2", "code-3"}, codes)
	})
}

func testCleanAppInstalls(t *testing.T, d TestDriver) {
	t.Run("permanently deletes the installations before the date", func(t *testing.T) {
		repo := newTestRepo(t, d)
		now := time.Now()
		SaveInstalls(t, repo,
			CompanyGrant("company-1", "old-company", now.Add(-48*time.Hour)),
			LocationAccess("company-1", "loc-1", "old-location", now.Add(-25*time.Hour)),
			CompanyGrant("company-1", "recent-company", now.Add(-1*time.Hour)),
			LocationAccess("company-1", "loc-1", "recent-location", now),
		)

		require.NoError(t, repo.CleanAppInstalls(now.Add(-24*time.Hour)))

		all, err := repo.FindAllAppInstalls()
		require.NoError(t, err)
		codes := make([]string, 0, len(all))
		for _, install := range all {
			codes = append(codes, install.AuthorizationCode)
		}
		assert.ElementsMatch(t, []string{"recent-company", "recent-location"}, codes)

		var count int64
		require.NoError(t, repo.coreDb.Unscoped().Model(&spec.DbGHLAppInstallation{}).Count(&count).Error)
		assert.EqualValues(t, 2, count, "the old installations must not be soft deleted")
	})

	t.Run("nothing to clean", func(t *testing.T) {
		repo := newTestRepo(t, d)
		now := time.Now()
		SaveInstalls(t, repo, CompanyGrant("company-1", "code-1", now))

		require.NoError(t, repo.CleanAppInstalls(now.Add(-time.Hour)))

		all, err := repo.FindAllAppInstalls()
		require.NoError(t, err)
		assert.Len(t, all, 1)
	})

	t.Run("empty table", func(t *testing.T) {
		repo := newTestRepo(t, d)
		assert.NoError(t, repo.CleanAppInstalls(time.Now()))
	})
}

func testFindInstalledLocations(t *testing.T, d TestDriver) {
	t.Run("locations of the company", func(t *testing.T) {
		repo := newTestRepo(t, d)
		SaveInstalls(t, repo, CompanyGrant("company-1", "code-1", time.Now()))
		require.NoError(t, repo.SaveInstalledLocation(&spec.DbGHLInstalledLocation{CompanyId: "company-1", LocationId: "loc-1", Name: "One"}))
		require.NoError(t, repo.SaveInstalledLocation(&spec.DbGHLInstalledLocation{CompanyId: "company-1", LocationId: "loc-2", Name: "Two"}))
		require.NoError(t, repo.SaveInstalledLocation(&spec.DbGHLInstalledLocation{CompanyId: "company-2", LocationId: "loc-3", Name: "Three"}))

		locations, err := repo.FindInstalledLocations("company-1")
		require.NoError(t, err)
		ids := make([]string, 0, len(locations))
		for _, location := range locations {
			ids = append(ids, location.LocationId)
		}
		assert.ElementsMatch(t, []string{"loc-1", "loc-2"}, ids)
	})

	t.Run("company with a grant and no locations", func(t *testing.T) {
		repo := newTestRepo(t, d)
		SaveInstalls(t, repo, CompanyGrant("company-1", "code-1", time.Now()))

		locations, err := repo.FindInstalledLocations("company-1")
		require.NoError(t, err)
		assert.Empty(t, locations)
	})

	t.Run("company without a grant", func(t *testing.T) {
		repo := newTestRepo(t, d)
		SaveInstalls(t, repo, LocationAccess("company-1", "loc-1", "code-1", time.Now()))
		require.NoError(t, repo.SaveInstalledLocation(&spec.DbGHLInstalledLocation{CompanyId: "company-1", LocationId: "loc-1"}))

		locations, err := repo.FindInstalledLocations("company-1")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, locations)
	})
}

func testFindInstalledLocation(t *testing.T, d TestDriver) {
	repo := newTestRepo(t, d)
	installedAt := time.Now().Add(-time.Hour).Truncate(time.Second)
	require.NoError(t, repo.SaveInstalledLocation(&spec.DbGHLInstalledLocation{
		CompanyId:   "company-1",
		LocationId:  "loc-1",
		Address:     "1 Main St",
		InstalledAt: &installedAt,
		IsInstalled: utils.PtrBool(true),
		Name:        "One",
		VersionId:   utils.PtrString("v1"),
	}))
	require.NoError(t, repo.SaveInstalledLocation(&spec.DbGHLInstalledLocation{CompanyId: "company-1", LocationId: "loc-2", Name: "Two"}))

	t.Run("finds by location id", func(t *testing.T) {
		location, err := repo.FindInstalledLocation("loc-1")
		require.NoError(t, err)
		assert.Equal(t, "company-1", location.CompanyId)
		assert.Equal(t, "1 Main St", location.Address)
		require.NotNil(t, location.InstalledAt)
		assert.True(t, installedAt.Equal(*location.InstalledAt))
		assert.True(t, *location.IsInstalled)
		assert.Equal(t, "One", location.Name)
		assert.Equal(t, "v1", *location.VersionId)
	})

	t.Run("unknown location", func(t *testing.T) {
		location, err := repo.FindInstalledLocation("loc-unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, location)
	})
}

func testSaveInstalledLocation(t *testing.T, d TestDriver) {
	countLocations := func(t *testing.T, repo *coreRepoImpl) int64 {
		t.Helper()
		var count int64
		require.NoError(t, repo.coreDb.Model(&spec.DbGHLInstalledLocation{}).Count(&count).Error)
		return count
	}

	t.Run("creates a new location", func(t *testing.T) {
		repo := newTestRepo(t, d)
		location := &spec.DbGHLInstalledLocation{CompanyId: "company-1", LocationId: "loc-1", Name: "One"}
		require.NoError(t, repo.SaveInstalledLocation(location))
		assert.NotZero(t, location.ID)
		assert.EqualValues(t, 1, countLocations(t, repo))
	})

	t.Run("a new record updates the location with the same id", func(t *testing.T) {
		repo := newTestRepo(t, d)
		first := &spec.DbGHLInstalledLocation{
			CompanyId:   "company-1",
			LocationId:  "loc-1",
			Name:        "Old",
			Address:     "Old address",
			IsInstalled: utils.PtrBool(true),
			VersionId:   utils.PtrString("v1"),
		}
		require.NoError(t, repo.SaveInstalledLocation(first))

		installedAt := time.Now().Truncate(time.Second)
		require.NoError(t, repo.SaveInstalledLocation(&spec.DbGHLInstalledLocation{
			CompanyId:   "company-2",
			LocationId:  "loc-1",
			Name:        "New",
			Address:     "New address",
			InstalledAt: &installedAt,
			IsInstalled: utils.PtrBool(false),
			VersionId:   utils.PtrString("v2"),
		}))

		assert.EqualValues(t, 1, countLocations(t, repo))
		found, err := repo.FindInstalledLocation("loc-1")
		require.NoError(t, err)
		assert.Equal(t, first.ID, found.ID)
		assert.Equal(t, "company-2", found.CompanyId)
		assert.Equal(t, "New", found.Name)
		assert.Equal(t, "New address", found.Address)
		require.NotNil(t, found.InstalledAt)
		assert.True(t, installedAt.Equal(*found.InstalledAt))
		assert.False(t, *found.IsInstalled)
		assert.Equal(t, "v2", *found.VersionId)
	})

	t.Run("a record with an id is saved as is", func(t *testing.T) {
		repo := newTestRepo(t, d)
		location := &spec.DbGHLInstalledLocation{CompanyId: "company-1", LocationId: "loc-1", Name: "Old"}
		require.NoError(t, repo.SaveInstalledLocation(location))

		location.Name = "Renamed"
		location.LocationId = "loc-2"
		require.NoError(t, repo.SaveInstalledLocation(location))

		assert.EqualValues(t, 1, countLocations(t, repo))
		found, err := repo.FindInstalledLocation("loc-2")
		require.NoError(t, err)
		assert.Equal(t, location.ID, found.ID)
		assert.Equal(t, "Renamed", found.Name)
		_, err = repo.FindInstalledLocation("loc-1")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("different locations are kept apart", func(t *testing.T) {
		repo := newTestRepo(t, d)
		require.NoError(t, repo.SaveInstalledLocation(&spec.DbGHLInstalledLocation{CompanyId: "company-1", LocationId: "loc-1"}))
		require.NoError(t, repo.SaveInstalledLocation(&spec.DbGHLInstalledLocation{CompanyId: "company-1", LocationId: "loc-2"}))
		assert.EqualValues(t, 2, countLocations(t, repo))
	})
}

func testFindServiceAccount(t *testing.T, d TestDriver) {
	repo := newTestRepo(t, d)
	require.NoError(t, repo.SaveServiceAccount(&spec.DbServiceAccount{ClientId: "client-1", ClientSecret: utils.PtrString("secret-1")}))
	require.NoError(t, repo.SaveServiceAccount(&spec.DbServiceAccount{ClientId: "client-2", ClientSecret: utils.PtrString("secret-2")}))

	t.Run("finds by client id", func(t *testing.T) {
		account, err := repo.FindServiceAccount(utils.PtrString("client-2"))
		require.NoError(t, err)
		assert.Equal(t, "client-2", account.ClientId)
		assert.Equal(t, "secret-2", *account.ClientSecret)
		assert.Equal(t, spec.LDActive, account.Status)
	})

	t.Run("unknown client id", func(t *testing.T) {
		account, err := repo.FindServiceAccount(utils.PtrString("client-unknown"))
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, account)
	})

	t.Run("nil client id", func(t *testing.T) {
		account, err := repo.FindServiceAccount(nil)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, account)
	})
}

func testSaveServiceAccount(t *testing.T, d TestDriver) {
	t.Run("updates an existing account", func(t *testing.T) {
		repo := newTestRepo(t, d)
		account := &spec.DbServiceAccount{ClientId: "client-1", ClientSecret: utils.PtrString("old")}
		require.NoError(t, repo.SaveServiceAccount(account))
		assert.NotZero(t, account.ID)

		account.ClientSecret = utils.PtrString("new")
		account.Status = spec.LDInactive
		require.NoError(t, repo.SaveServiceAccount(account))

		found, err := repo.FindServiceAccount(utils.PtrString("client-1"))
		require.NoError(t, err)
		assert.Equal(t, account.ID, found.ID)
		assert.Equal(t, "new", *found.ClientSecret)
		assert.Equal(t, spec.LDInactive, found.Status)
	})

	t.Run("duplicated client id", func(t *testing.T) {
		repo := newTestRepo(t, d)
		require.NoError(t, repo.SaveServiceAccount(&spec.DbServiceAccount{ClientId: "client-1"}))
		assert.Error(t, repo.SaveServiceAccount(&spec.DbServiceAccount{ClientId: "client-1"}))
	})
}

func testFindNode(t *testing.T, d TestDriver) {
	repo := newTestRepo(t, d)
	heartbeat := time.Now().Truncate(time.Second)
	require.NoError(t, repo.SaveNode(&spec.DbServerNode{NodeId: "node-1", Status: spec.Live, LastHeartbeat: &heartbeat}))
	require.NoError(t, repo.SaveNode(&spec.DbServerNode{NodeId: "node-2", Status: spec.Starting}))

	t.Run("finds by node id", func(t *testing.T) {
		node, err := repo.FindNode("node-1")
		require.NoError(t, err)
		assert.Equal(t, "node-1", node.NodeId)
		assert.Equal(t, spec.Live, node.Status)
		require.NotNil(t, node.LastHeartbeat)
		assert.True(t, heartbeat.Equal(*node.LastHeartbeat))
	})

	t.Run("unknown node id", func(t *testing.T) {
		node, err := repo.FindNode("node-unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, node)
	})
}

func testFindAllNodes(t *testing.T, d TestDriver) {
	t.Run("no nodes", func(t *testing.T) {
		repo := newTestRepo(t, d)
		nodes, err := repo.FindAllNodes()
		require.NoError(t, err)
		assert.Empty(t, nodes)
	})

	t.Run("all the nodes", func(t *testing.T) {
		repo := newTestRepo(t, d)
		require.NoError(t, repo.SaveNode(&spec.DbServerNode{NodeId: "node-1", Status: spec.Live}))
		require.NoError(t, repo.SaveNode(&spec.DbServerNode{NodeId: "node-2"}))

		nodes, err := repo.FindAllNodes()
		require.NoError(t, err)
		statuses := make(map[string]spec.LDServerNodeStatus, len(nodes))
		for _, node := range nodes {
			statuses[node.NodeId] = node.Status
		}
		assert.Equal(t, map[string]spec.LDServerNodeStatus{"node-1": spec.Live, "node-2": spec.Down}, statuses)
	})
}

func testSaveNode(t *testing.T, d TestDriver) {
	t.Run("a new record updates the node with the same id", func(t *testing.T) {
		repo := newTestRepo(t, d)
		first := &spec.DbServerNode{NodeId: "node-1", Status: spec.Starting}
		require.NoError(t, repo.SaveNode(first))
		assert.NotZero(t, first.ID)

		heartbeat := time.Now().Truncate(time.Second)
		second := &spec.DbServerNode{NodeId: "node-1", Status: spec.Live, LastHeartbeat: &heartbeat}
		require.NoError(t, repo.SaveNode(second))
		assert.Equal(t, first.ID, second.ID)

		nodes, err := repo.FindAllNodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		assert.Equal(t, first.ID, nodes[0].ID)
		assert.Equal(t, spec.Live, nodes[0].Status)
		require.NotNil(t, nodes[0].LastHeartbeat)
		assert.True(t, heartbeat.Equal(*nodes[0].LastHeartbeat))
		// postgres keeps the times with microseconds
		assert.WithinDuration(t, first.CreatedAt, nodes[0].CreatedAt, time.Microsecond)
	})

	t.Run("a record with an id is saved as is", func(t *testing.T) {
		repo := newTestRepo(t, d)
		node := &spec.DbServerNode{NodeId: "node-1", Status: spec.Live}
		require.NoError(t, repo.SaveNode(node))

		node.Status = spec.ShutingDown
		require.NoError(t, repo.SaveNode(node))

		nodes, err := repo.FindAllNodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		assert.Equal(t, spec.ShutingDown, nodes[0].Status)
	})

	t.Run("different nodes are kept apart", func(t *testing.T) {
		repo := newTestRepo(t, d)
		require.NoError(t, repo.SaveNode(&spec.DbServerNode{NodeId: "node-1"}))
		require.NoError(t, repo.SaveNode(&spec.DbServerNode{NodeId: "node-2"}))

		nodes, err := repo.FindAllNodes()
		require.NoError(t, err)
		assert.Len(t, nodes, 2)
	})
}

func testConcurrentSaves(t *testing.T, d TestDriver) {
	if d.Cfg.Driver == spec.SqliteDriver && d.Cfg.InMemorySQLite {
		t.Skip("the shared cache of in-memory sqlite fails concurrent writes with table locked errors")
	}

	t.Run("new records with the same id keep one record", func(t *testing.T) {
		repo := newTestRepo(t, d)
		var wg sync.WaitGroup
		errs := make(chan error, 20)
		for i := 0; i < 10; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				errs <- repo.SaveNode(&spec.DbServerNode{NodeId: "node-1", Status: spec.Live})
			}()
			go func() {
				defer wg.Done()
				errs <- repo.SaveInstalledLocation(&spec.DbGHLInstalledLocation{CompanyId: "company-1", LocationId: "loc-1"})
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			assert.NoError(t, err)
		}

		var nodes, locations int64
		require.NoError(t, repo.coreDb.Model(&spec.DbServerNode{}).Count(&nodes).Error)
		require.NoError(t, repo.coreDb.Model(&spec.DbGHLInstalledLocation{}).Count(&locations).Error)
		assert.EqualValues(t, 1, nodes)
		assert.EqualValues(t, 1, locations)
	})

	t.Run("a new record restores the deleted one with the same id", func(t *testing.T) {
		repo := newTestRepo(t, d)
		first := &spec.DbServerNode{NodeId: "node-1", Status: spec.Live}
		require.NoError(t, repo.SaveNode(first))
		require.NoError(t, repo.coreDb.Delete(first).Error)

		second := &spec.DbServerNode{NodeId: "node-1", Status: spec.Starting}
		require.NoError(t, repo.SaveNode(second))

		found, err := repo.FindNode("node-1")
		require.NoError(t, err)
		assert.Equal(t, first.ID, found.ID)
		assert.Equal(t, spec.Starting, found.Status)
	})
}

func TestIsRetryable(t *testing.T) {
	assert.True(t, isRetryable(&pgconn.PgError{Code: "40001"}))
	assert.True(t, isRetryable(fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: "40001"})))
	assert.False(t, isRetryable(&pgconn.PgError{Code: "23505"}))
	assert.False(t, isRetryable(gorm.ErrRecordNotFound))
	assert.False(t, isRetryable(nil))
}
