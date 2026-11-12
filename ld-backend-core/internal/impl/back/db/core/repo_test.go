package ldcore_test

import (
	ldconn "ld-backend/internal/impl/back/db/conn"
	"ld-backend/internal/impl/back/db/core"
	ldutils "ld-backend/internal/impl/core/utils"
	"ld-backend/internal/spec"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	ldconn.RunTestsWithDbs(m)
}

func testMigrations(t *testing.T, d ldconn.TestDriver) {
	t.Run("migrates the core tables", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		migrator := repo.Db().Migrator()
		for _, table := range ldspec.DbCoreTables {
			assert.True(t, migrator.HasTable(table))
		}
	})

	t.Run("migrates by the schema version", func(t *testing.T) {
		cfg := ldcore.NewTestDBConfig(t, d)
		connect := func(t *testing.T) ldcore.TestRepo {
			t.Helper()
			return ldcore.ConnectTestRepo(t, cfg)
		}
		setVersion := func(t *testing.T, repo ldcore.TestRepo, version string) {
			t.Helper()
			require.NoError(t, repo.Db().Create(&ldspec.DbSchemaVersion{Version: version}).Error)
		}

		repo := connect(t)
		version, err := ldconn.SchemaVersion(repo.Db())
		require.NoError(t, err)
		assert.Equal(t, ldspec.DB_SCHEMA_VERSION, version, "a new database is migrated")

		// a table dropped stays dropped while the version is current
		require.NoError(t, repo.Db().Migrator().DropTable(&ldspec.DbServerNode{}))
		repo = connect(t)
		assert.False(t, repo.Db().Migrator().HasTable(&ldspec.DbServerNode{}), "current version is not migrated")

		setVersion(t, repo, "99.0.0")
		repo = connect(t)
		assert.False(t, repo.Db().Migrator().HasTable(&ldspec.DbServerNode{}), "newer version is not migrated")
		version, err = ldconn.SchemaVersion(repo.Db())
		require.NoError(t, err)
		assert.Equal(t, "99.0.0", version, "newer version is kept")

		setVersion(t, repo, "0.0.1")
		repo = connect(t)
		assert.True(t, repo.Db().Migrator().HasTable(&ldspec.DbServerNode{}), "older version is migrated")
		version, err = ldconn.SchemaVersion(repo.Db())
		require.NoError(t, err)
		assert.Equal(t, ldspec.DB_SCHEMA_VERSION, version)
	})

	t.Run("migrates an existing database again", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.Db().AutoMigrate(ldspec.DbCoreTables...))
	})
}

// TestCoreRepo runs all the repo cases with each database driver.
func TestCoreRepo(t *testing.T) {
	for _, d := range ldconn.TestDrivers() {
		t.Run(d.Name, func(t *testing.T) {
			d.Check(t)

			t.Run("Migrations", func(t *testing.T) {
				testMigrations(t, d)
			})

			t.Run("FindTenant", func(t *testing.T) {
				testFindTenant(t, d)
			})

			t.Run("SaveTenant", func(t *testing.T) {
				testSaveTenant(t, d)
			})

			t.Run("FindCompanies", func(t *testing.T) {
				testFindCompanies(t, d)
			})

			t.Run("FindCompany", func(t *testing.T) {
				testFindCompany(t, d)
			})

			t.Run("SaveCompany", func(t *testing.T) {
				testSaveCompany(t, d)
			})

			t.Run("SaveCompanyAppInstall", func(t *testing.T) {
				testSaveCompanyAppInstall(t, d)
			})

			t.Run("FindLocations", func(t *testing.T) {
				testFindLocations(t, d)
			})

			t.Run("FindLocation", func(t *testing.T) {
				testFindLocation(t, d)
			})

			t.Run("SaveLocation", func(t *testing.T) {
				testSaveLocation(t, d)
			})

			t.Run("SaveLocationAppInstall", func(t *testing.T) {
				testSaveLocationAppInstall(t, d)
			})

			t.Run("FindServiceAccount", func(t *testing.T) {
				testFindServiceAccount(t, d)
			})

			t.Run("SaveServiceAccount", func(t *testing.T) {
				testSaveServiceAccount(t, d)
			})

			t.Run("FindNode", func(t *testing.T) {
				testFindNode(t, d)
			})

			t.Run("FindAllNodes", func(t *testing.T) {
				testFindAllNodes(t, d)
			})

			t.Run("SaveNode", func(t *testing.T) {
				testSaveNode(t, d)
			})

			t.Run("FindPayments", func(t *testing.T) {
				testFindPayments(t, d)
			})

			t.Run("SavePayment", func(t *testing.T) {
				testSavePayment(t, d)
			})

			t.Run("FindPaidSubscription", func(t *testing.T) {
				testFindPaidSubscription(t, d)
			})

			t.Run("SavePaidSubscription", func(t *testing.T) {
				testSavePaidSubscription(t, d)
			})

			t.Run("ConcurrentSaves", func(t *testing.T) {
				testConcurrentSaves(t, d)
			})
		})
	}
}

func TestNewCoreRepo(t *testing.T) {
	t.Run("unsupported driver", func(t *testing.T) {
		dbCfg := ldspec.DBConfig{
			Driver:     "mysql",
			CoreDBName: "",
			DbUrlTpl:   "some-url",
		}
		repo, err := ldcore.NewCoreRepo(dbCfg)
		require.Error(t, err)
		assert.Nil(t, repo)
	})
}

func testFindTenant(t *testing.T, d ldconn.TestDriver) {
	repo := ldcore.NewTestRepo(t, d)
	require.NoError(t, repo.SaveTenant(&ldspec.DbTenant{BusinessId: "loc-1", DbName: "loc-1", Title: "Location 1"}))
	require.NoError(t, repo.SaveTenant(&ldspec.DbTenant{BusinessId: "loc-2", DbName: "loc-2", Title: "Location 2"}))

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

func testSaveTenant(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates with the default status", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		tenant := &ldspec.DbTenant{
			BusinessId: "loc-1",
			DbName:     "loc-1",
			Title:      "Location 1",
			Url:        "file:loc-1",
			Driver:     string(ldspec.SqliteDriver),
		}
		require.NoError(t, repo.SaveTenant(tenant))
		assert.NotZero(t, tenant.ID)

		found, err := repo.FindTenant("loc-1")
		require.NoError(t, err)
		assert.Equal(t, tenant.ID, found.ID)
		assert.Equal(t, ldspec.LDActive, found.Status)
		assert.Equal(t, "file:loc-1", found.Url)
		assert.Equal(t, string(ldspec.SqliteDriver), found.Driver)
	})

	t.Run("updates an existing tenant", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		tenant := &ldspec.DbTenant{BusinessId: "loc-1", Title: "Old", Status: ldspec.LDActive}
		require.NoError(t, repo.SaveTenant(tenant))

		tenant.Title = "New"
		tenant.Status = ldspec.LDInactive
		require.NoError(t, repo.SaveTenant(tenant))

		found, err := repo.FindTenant("loc-1")
		require.NoError(t, err)
		assert.Equal(t, tenant.ID, found.ID)
		assert.Equal(t, "New", found.Title)
		assert.Equal(t, ldspec.LDInactive, found.Status)

		var count int64
		require.NoError(t, repo.Db().Model(&ldspec.DbTenant{}).Count(&count).Error)
		assert.EqualValues(t, 1, count)
	})
}

// authCodePtr is the optional authorization code; nil when empty.
func authCodePtr(authCode string) *string {
	if authCode == "" {
		return nil
	}
	return &authCode
}

// company is a GHL company installed with the authorization code.
func company(companyId string, authCode string) *ldspec.DbGHLCompany {
	return &ldspec.DbGHLCompany{
		CompanyId:       companyId,
		Name:            "Company " + companyId,
		InstallAuthCode: authCodePtr(authCode),
		AccessToken:     ldutils.Ptr("access-" + authCode),
		RefreshToken:    ldutils.Ptr("refresh-" + authCode),
		UserId:          ldutils.Ptr("user-" + companyId),
	}
}

// location is a GHL location of the company installed with the
// authorization code.
func location(companyId string, locationId string, authCode string) *ldspec.DbGHLLocation {
	return &ldspec.DbGHLLocation{
		CompanyId:       companyId,
		LocationId:      locationId,
		Name:            "Location " + locationId,
		AccessToken:     ldutils.Ptr("access-" + authCode),
		RefreshToken:    ldutils.Ptr("refresh-" + authCode),
		InstallType:     ldspec.DbGHLInstallTypeLocation,
		UserId:          ldutils.Ptr("user-" + locationId),
		InstallAuthCode: authCodePtr(authCode),
	}
}

func countCompanies(t *testing.T, repo ldcore.TestRepo) int64 {
	t.Helper()
	var count int64
	require.NoError(t, repo.Db().Model(&ldspec.DbGHLCompany{}).Count(&count).Error)
	return count
}

func countLocations(t *testing.T, repo ldcore.TestRepo) int64 {
	t.Helper()
	var count int64
	require.NoError(t, repo.Db().Model(&ldspec.DbGHLLocation{}).Count(&count).Error)
	return count
}

func testFindCompanies(t *testing.T, d ldconn.TestDriver) {
	t.Run("no companies", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		companies, err := repo.FindCompanies()
		require.NoError(t, err)
		assert.NotNil(t, companies)
		assert.Empty(t, companies)
	})

	t.Run("all the companies", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.SaveCompany(company("company-1", "code-1")))
		require.NoError(t, repo.SaveCompany(company("company-2", "code-2")))

		companies, err := repo.FindCompanies()
		require.NoError(t, err)
		ids := make([]string, 0, len(companies))
		for _, c := range companies {
			ids = append(ids, c.CompanyId)
		}
		assert.ElementsMatch(t, []string{"company-1", "company-2"}, ids)
	})

	t.Run("ignores deleted companies", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		deleted := company("company-1", "code-1")
		require.NoError(t, repo.SaveCompany(deleted))
		require.NoError(t, repo.SaveCompany(company("company-2", "code-2")))
		require.NoError(t, repo.Db().Delete(deleted).Error)

		companies, err := repo.FindCompanies()
		require.NoError(t, err)
		require.Len(t, companies, 1)
		assert.Equal(t, "company-2", companies[0].CompanyId)
	})
}

func testFindCompany(t *testing.T, d ldconn.TestDriver) {
	repo := ldcore.NewTestRepo(t, d)
	require.NoError(t, repo.SaveCompany(company("company-1", "code-1")))
	require.NoError(t, repo.SaveCompany(company("company-2", "code-2")))

	t.Run("the company with the id", func(t *testing.T) {
		found, err := repo.FindCompany("company-1")
		require.NoError(t, err)
		assert.Equal(t, "company-1", found.CompanyId)
		assert.Equal(t, "Company company-1", found.Name)
		assert.Equal(t, ldutils.Ptr("code-1"), found.InstallAuthCode)
		assert.Equal(t, ldutils.Ptr("access-code-1"), found.AccessToken)
		assert.Equal(t, ldutils.Ptr("refresh-code-1"), found.RefreshToken)
		assert.Equal(t, ldutils.Ptr("user-company-1"), found.UserId)
	})

	t.Run("unknown company", func(t *testing.T) {
		found, err := repo.FindCompany("company-unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, found)
	})
}

func testSaveCompany(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates a new company", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		c := company("company-1", "code-1")
		require.NoError(t, repo.SaveCompany(c))
		assert.NotZero(t, c.ID)
		assert.EqualValues(t, 1, countCompanies(t, repo))
	})

	t.Run("a new record updates the company with the same id", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first := company("company-1", "code-1")
		require.NoError(t, repo.SaveCompany(first))

		second := company("company-1", "code-2")
		second.Name = "Renamed"
		second.UserId = ldutils.Ptr("user-2")
		second.ExpiresIn = ldutils.Ptr(int64(3600))
		require.NoError(t, repo.SaveCompany(second))
		assert.Equal(t, first.ID, second.ID)

		assert.EqualValues(t, 1, countCompanies(t, repo))
		found, err := repo.FindCompany("company-1")
		require.NoError(t, err)
		assert.Equal(t, first.ID, found.ID)
		assert.Equal(t, "Renamed", found.Name)
		assert.Equal(t, ldutils.Ptr("code-2"), found.InstallAuthCode)
		assert.Equal(t, ldutils.Ptr("access-code-2"), found.AccessToken)
		assert.Equal(t, ldutils.Ptr("refresh-code-2"), found.RefreshToken)
		assert.Equal(t, ldutils.Ptr("user-2"), found.UserId)
		assert.Equal(t, ldutils.Ptr(int64(3600)), found.ExpiresIn)
	})

	t.Run("a new record restores the deleted company with the same id", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first := company("company-1", "code-1")
		require.NoError(t, repo.SaveCompany(first))
		require.NoError(t, repo.Db().Delete(first).Error)

		require.NoError(t, repo.SaveCompany(company("company-1", "code-2")))

		found, err := repo.FindCompany("company-1")
		require.NoError(t, err)
		assert.Equal(t, first.ID, found.ID)
		assert.Equal(t, ldutils.Ptr("code-2"), found.InstallAuthCode)
	})

	t.Run("a record with an id is saved as is", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		c := company("company-1", "code-1")
		require.NoError(t, repo.SaveCompany(c))

		c.AccessToken = ldutils.Ptr("refreshed")
		c.CompanyId = "company-2"
		require.NoError(t, repo.SaveCompany(c))

		assert.EqualValues(t, 1, countCompanies(t, repo))
		found, err := repo.FindCompany("company-2")
		require.NoError(t, err)
		assert.Equal(t, c.ID, found.ID)
		assert.Equal(t, ldutils.Ptr("refreshed"), found.AccessToken)
		_, err = repo.FindCompany("company-1")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("different companies are kept apart", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.SaveCompany(company("company-1", "code-1")))
		require.NoError(t, repo.SaveCompany(company("company-2", "code-2")))
		assert.EqualValues(t, 2, countCompanies(t, repo))
	})
}

func testSaveCompanyAppInstall(t *testing.T, d ldconn.TestDriver) {
	t.Run("saves the company with the authorization code", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		c := company("company-1", "")
		saved, err := repo.SaveCompanyAppInstall("code-1", c)
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.NotZero(t, saved.ID)
		assert.Equal(t, ldutils.Ptr("code-1"), saved.InstallAuthCode)

		found, err := repo.FindCompany("company-1")
		require.NoError(t, err)
		assert.Equal(t, saved.ID, found.ID)
		assert.Equal(t, ldutils.Ptr("code-1"), found.InstallAuthCode)
	})

	t.Run("an authorization code already used returns the saved company unchanged", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first, err := repo.SaveCompanyAppInstall("code-1", company("company-1", ""))
		require.NoError(t, err)

		again := company("company-1", "")
		again.AccessToken = ldutils.Ptr("other-access")
		saved, err := repo.SaveCompanyAppInstall("code-1", again)
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.Equal(t, first.ID, saved.ID)
		assert.Equal(t, first.AccessToken, saved.AccessToken)

		assert.EqualValues(t, 1, countCompanies(t, repo))
		found, err := repo.FindCompany("company-1")
		require.NoError(t, err)
		assert.Equal(t, first.AccessToken, found.AccessToken)
	})

	t.Run("a new authorization code updates the company with the same id", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first, err := repo.SaveCompanyAppInstall("code-1", company("company-1", "code-1"))
		require.NoError(t, err)

		saved, err := repo.SaveCompanyAppInstall("code-2", company("company-1", "code-2"))
		require.NoError(t, err)
		assert.Equal(t, first.ID, saved.ID)

		assert.EqualValues(t, 1, countCompanies(t, repo))
		found, err := repo.FindCompany("company-1")
		require.NoError(t, err)
		assert.Equal(t, ldutils.Ptr("code-2"), found.InstallAuthCode)
		assert.Equal(t, ldutils.Ptr("access-code-2"), found.AccessToken)
	})
}

func testFindLocations(t *testing.T, d ldconn.TestDriver) {
	t.Run("the locations of the company", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.SaveLocation(location("company-1", "loc-1", "code-1")))
		require.NoError(t, repo.SaveLocation(location("company-1", "loc-2", "code-2")))
		require.NoError(t, repo.SaveLocation(location("company-2", "loc-3", "code-3")))

		locations, err := repo.FindLocations("company-1")
		require.NoError(t, err)
		ids := make([]string, 0, len(locations))
		for _, l := range locations {
			ids = append(ids, l.LocationId)
		}
		assert.ElementsMatch(t, []string{"loc-1", "loc-2"}, ids)
	})

	t.Run("unknown company", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.SaveLocation(location("company-1", "loc-1", "code-1")))
		locations, err := repo.FindLocations("company-unknown")
		require.NoError(t, err)
		assert.NotNil(t, locations)
		assert.Empty(t, locations)
	})

	t.Run("ignores deleted locations", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		deleted := location("company-1", "loc-1", "code-1")
		require.NoError(t, repo.SaveLocation(deleted))
		require.NoError(t, repo.SaveLocation(location("company-1", "loc-2", "code-2")))
		require.NoError(t, repo.Db().Delete(deleted).Error)

		locations, err := repo.FindLocations("company-1")
		require.NoError(t, err)
		require.Len(t, locations, 1)
		assert.Equal(t, "loc-2", locations[0].LocationId)
	})
}

func testFindLocation(t *testing.T, d ldconn.TestDriver) {
	repo := ldcore.NewTestRepo(t, d)
	require.NoError(t, repo.SaveLocation(location("company-1", "loc-1", "code-1")))
	require.NoError(t, repo.SaveLocation(location("company-1", "loc-2", "code-2")))

	t.Run("the location with the id", func(t *testing.T) {
		found, err := repo.FindLocation("loc-1")
		require.NoError(t, err)
		assert.Equal(t, "company-1", found.CompanyId)
		assert.Equal(t, "loc-1", found.LocationId)
		assert.Equal(t, "Location loc-1", found.Name)
		assert.Equal(t, ldutils.Ptr("access-code-1"), found.AccessToken)
		assert.Equal(t, ldutils.Ptr("refresh-code-1"), found.RefreshToken)
		assert.EqualValues(t, ldspec.DbGHLInstallTypeLocation, found.InstallType)
		assert.Equal(t, ldutils.Ptr("user-loc-1"), found.UserId)
		assert.Equal(t, ldutils.Ptr("code-1"), found.InstallAuthCode)
	})

	t.Run("unknown location", func(t *testing.T) {
		found, err := repo.FindLocation("loc-unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, found)
	})
}

func testSaveLocation(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates a new location", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		l := location("company-1", "loc-1", "code-1")
		require.NoError(t, repo.SaveLocation(l))
		assert.NotZero(t, l.ID)
		assert.EqualValues(t, 1, countLocations(t, repo))
	})

	t.Run("a new record updates the location with the same id", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first := location("company-1", "loc-1", "code-1")
		require.NoError(t, repo.SaveLocation(first))

		second := location("company-2", "loc-1", "code-2")
		second.Name = "Renamed"
		second.InstallType = ldspec.DbGHLInstallTypeCompany
		second.UserId = ldutils.Ptr("user-2")
		second.ExpiresIn = ldutils.Ptr(int64(3600))
		require.NoError(t, repo.SaveLocation(second))
		assert.Equal(t, first.ID, second.ID)

		assert.EqualValues(t, 1, countLocations(t, repo))
		found, err := repo.FindLocation("loc-1")
		require.NoError(t, err)
		assert.Equal(t, first.ID, found.ID)
		assert.Equal(t, "company-2", found.CompanyId)
		assert.Equal(t, "Renamed", found.Name)
		assert.Equal(t, ldutils.Ptr("access-code-2"), found.AccessToken)
		assert.Equal(t, ldutils.Ptr("refresh-code-2"), found.RefreshToken)
		assert.EqualValues(t, ldspec.DbGHLInstallTypeCompany, found.InstallType)
		assert.Equal(t, ldutils.Ptr("user-2"), found.UserId)
		assert.Equal(t, ldutils.Ptr("code-2"), found.InstallAuthCode)
		assert.Equal(t, ldutils.Ptr(int64(3600)), found.ExpiresIn)
	})

	t.Run("a new record restores the deleted location with the same id", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first := location("company-1", "loc-1", "code-1")
		require.NoError(t, repo.SaveLocation(first))
		require.NoError(t, repo.Db().Delete(first).Error)

		require.NoError(t, repo.SaveLocation(location("company-1", "loc-1", "code-2")))

		found, err := repo.FindLocation("loc-1")
		require.NoError(t, err)
		assert.Equal(t, first.ID, found.ID)
		assert.Equal(t, ldutils.Ptr("code-2"), found.InstallAuthCode)
	})

	t.Run("a record with an id is saved as is", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		l := location("company-1", "loc-1", "code-1")
		require.NoError(t, repo.SaveLocation(l))

		l.Name = "Renamed"
		l.LocationId = "loc-2"
		require.NoError(t, repo.SaveLocation(l))

		assert.EqualValues(t, 1, countLocations(t, repo))
		found, err := repo.FindLocation("loc-2")
		require.NoError(t, err)
		assert.Equal(t, l.ID, found.ID)
		assert.Equal(t, "Renamed", found.Name)
		_, err = repo.FindLocation("loc-1")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("different locations are kept apart", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.SaveLocation(location("company-1", "loc-1", "code-1")))
		require.NoError(t, repo.SaveLocation(location("company-1", "loc-2", "code-2")))
		assert.EqualValues(t, 2, countLocations(t, repo))
	})

	t.Run("many locations installed by the company have no authorization code", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		for _, id := range []string{"loc-1", "loc-2"} {
			l := location("company-1", id, "")
			l.InstallType = ldspec.DbGHLInstallTypeCompany
			require.NoError(t, repo.SaveLocation(l))
		}
		assert.EqualValues(t, 2, countLocations(t, repo))
	})
}

func testSaveLocationAppInstall(t *testing.T, d ldconn.TestDriver) {
	t.Run("saves the location with the authorization code", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		saved, err := repo.SaveLocationAppInstall("code-1", location("company-1", "loc-1", ""))
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.NotZero(t, saved.ID)
		assert.Equal(t, ldutils.Ptr("code-1"), saved.InstallAuthCode)

		found, err := repo.FindLocation("loc-1")
		require.NoError(t, err)
		assert.Equal(t, saved.ID, found.ID)
		assert.Equal(t, ldutils.Ptr("code-1"), found.InstallAuthCode)
	})

	t.Run("an authorization code already used returns the saved location unchanged", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first, err := repo.SaveLocationAppInstall("code-1", location("company-1", "loc-1", ""))
		require.NoError(t, err)

		again := location("company-1", "loc-1", "")
		again.AccessToken = ldutils.Ptr("other-access")
		saved, err := repo.SaveLocationAppInstall("code-1", again)
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.Equal(t, first.ID, saved.ID)
		assert.Equal(t, first.AccessToken, saved.AccessToken)

		assert.EqualValues(t, 1, countLocations(t, repo))
		found, err := repo.FindLocation("loc-1")
		require.NoError(t, err)
		assert.Equal(t, first.AccessToken, found.AccessToken)
	})

	t.Run("a new authorization code updates the location with the same id", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first, err := repo.SaveLocationAppInstall("code-1", location("company-1", "loc-1", "code-1"))
		require.NoError(t, err)

		saved, err := repo.SaveLocationAppInstall("code-2", location("company-1", "loc-1", "code-2"))
		require.NoError(t, err)
		assert.Equal(t, first.ID, saved.ID)

		assert.EqualValues(t, 1, countLocations(t, repo))
		found, err := repo.FindLocation("loc-1")
		require.NoError(t, err)
		assert.Equal(t, ldutils.Ptr("code-2"), found.InstallAuthCode)
		assert.Equal(t, ldutils.Ptr("access-code-2"), found.AccessToken)
	})
}

func testFindServiceAccount(t *testing.T, d ldconn.TestDriver) {
	repo := ldcore.NewTestRepo(t, d)
	require.NoError(t, repo.SaveServiceAccount(&ldspec.DbServiceAccount{ClientId: "client-1", ClientSecret: ldutils.Ptr("secret-1")}))
	require.NoError(t, repo.SaveServiceAccount(&ldspec.DbServiceAccount{ClientId: "client-2", ClientSecret: ldutils.Ptr("secret-2")}))

	t.Run("finds by client id", func(t *testing.T) {
		account, err := repo.FindServiceAccount(ldutils.Ptr("client-2"))
		require.NoError(t, err)
		assert.Equal(t, "client-2", account.ClientId)
		assert.Equal(t, "secret-2", *account.ClientSecret)
		assert.Equal(t, ldspec.LDActive, account.Status)
	})

	t.Run("unknown client id", func(t *testing.T) {
		account, err := repo.FindServiceAccount(ldutils.Ptr("client-unknown"))
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, account)
	})

	t.Run("nil client id", func(t *testing.T) {
		account, err := repo.FindServiceAccount(nil)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, account)
	})
}

func testSaveServiceAccount(t *testing.T, d ldconn.TestDriver) {
	t.Run("updates an existing account", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		account := &ldspec.DbServiceAccount{ClientId: "client-1", ClientSecret: ldutils.Ptr("old")}
		require.NoError(t, repo.SaveServiceAccount(account))
		assert.NotZero(t, account.ID)

		account.ClientSecret = ldutils.Ptr("new")
		account.Status = ldspec.LDInactive
		require.NoError(t, repo.SaveServiceAccount(account))

		found, err := repo.FindServiceAccount(ldutils.Ptr("client-1"))
		require.NoError(t, err)
		assert.Equal(t, account.ID, found.ID)
		assert.Equal(t, "new", *found.ClientSecret)
		assert.Equal(t, ldspec.LDInactive, found.Status)
	})

	t.Run("duplicated client id", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.SaveServiceAccount(&ldspec.DbServiceAccount{ClientId: "client-1"}))
		assert.Error(t, repo.SaveServiceAccount(&ldspec.DbServiceAccount{ClientId: "client-1"}))
	})
}

func testFindNode(t *testing.T, d ldconn.TestDriver) {
	repo := ldcore.NewTestRepo(t, d)
	heartbeat := time.Now().Truncate(time.Second)
	require.NoError(t, repo.SaveNode(&ldspec.DbServerNode{NodeId: "node-1", Status: ldspec.Live, LastHeartbeat: &heartbeat}))
	require.NoError(t, repo.SaveNode(&ldspec.DbServerNode{NodeId: "node-2", Status: ldspec.Starting}))

	t.Run("finds by node id", func(t *testing.T) {
		node, err := repo.FindNode("node-1")
		require.NoError(t, err)
		assert.Equal(t, "node-1", node.NodeId)
		assert.Equal(t, ldspec.Live, node.Status)
		require.NotNil(t, node.LastHeartbeat)
		assert.True(t, heartbeat.Equal(*node.LastHeartbeat))
	})

	t.Run("unknown node id", func(t *testing.T) {
		node, err := repo.FindNode("node-unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, node)
	})
}

func testFindAllNodes(t *testing.T, d ldconn.TestDriver) {
	t.Run("no nodes", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		nodes, err := repo.FindAllNodes()
		require.NoError(t, err)
		assert.Empty(t, nodes)
	})

	t.Run("all the nodes", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.SaveNode(&ldspec.DbServerNode{NodeId: "node-1", Status: ldspec.Live}))
		require.NoError(t, repo.SaveNode(&ldspec.DbServerNode{NodeId: "node-2"}))

		nodes, err := repo.FindAllNodes()
		require.NoError(t, err)
		statuses := make(map[string]ldspec.LDServerNodeStatus, len(nodes))
		for _, node := range nodes {
			statuses[node.NodeId] = node.Status
		}
		assert.Equal(t, map[string]ldspec.LDServerNodeStatus{"node-1": ldspec.Live, "node-2": ldspec.Down}, statuses)
	})
}

func testSaveNode(t *testing.T, d ldconn.TestDriver) {
	t.Run("a new record updates the node with the same id", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first := &ldspec.DbServerNode{NodeId: "node-1", Status: ldspec.Starting}
		require.NoError(t, repo.SaveNode(first))
		assert.NotZero(t, first.ID)

		heartbeat := time.Now().Truncate(time.Second)
		second := &ldspec.DbServerNode{NodeId: "node-1", Status: ldspec.Live, LastHeartbeat: &heartbeat}
		require.NoError(t, repo.SaveNode(second))
		assert.Equal(t, first.ID, second.ID)

		nodes, err := repo.FindAllNodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		assert.Equal(t, first.ID, nodes[0].ID)
		assert.Equal(t, ldspec.Live, nodes[0].Status)
		require.NotNil(t, nodes[0].LastHeartbeat)
		assert.True(t, heartbeat.Equal(*nodes[0].LastHeartbeat))
		// postgres keeps the times with microseconds
		assert.WithinDuration(t, first.CreatedAt, nodes[0].CreatedAt, time.Microsecond)
	})

	t.Run("a record with an id is saved as is", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		node := &ldspec.DbServerNode{NodeId: "node-1", Status: ldspec.Live}
		require.NoError(t, repo.SaveNode(node))

		node.Status = ldspec.ShutingDown
		require.NoError(t, repo.SaveNode(node))

		nodes, err := repo.FindAllNodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		assert.Equal(t, ldspec.ShutingDown, nodes[0].Status)
	})

	t.Run("different nodes are kept apart", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.SaveNode(&ldspec.DbServerNode{NodeId: "node-1"}))
		require.NoError(t, repo.SaveNode(&ldspec.DbServerNode{NodeId: "node-2"}))

		nodes, err := repo.FindAllNodes()
		require.NoError(t, err)
		assert.Len(t, nodes, 2)
	})
}

func payment(companyId string, locationId string, amount float64, at time.Time) *ldspec.DbPaymentRecord {
	return &ldspec.DbPaymentRecord{
		BusinessId: ldutils.Ptr(companyId),
		LocationId: ldutils.Ptr(locationId),
		Amount:     &amount,
		DateTime:   &at,
	}
}

func testFindPayments(t *testing.T, d ldconn.TestDriver) {
	t.Run("returns the payments of the location, the newest first", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		now := time.Now().Truncate(time.Second)
		require.NoError(t, repo.SavePayment(payment("company-1", "loc-1", 10, now.Add(-2*time.Hour))))
		require.NoError(t, repo.SavePayment(payment("company-1", "loc-1", 30, now)))
		require.NoError(t, repo.SavePayment(payment("company-1", "loc-2", 50, now)))
		require.NoError(t, repo.SavePayment(payment("company-2", "loc-1", 20, now.Add(-time.Hour))))

		payments, err := repo.FindPayments("loc-1")
		require.NoError(t, err)
		require.Len(t, payments, 3)
		assert.Equal(t, 30.0, *payments[0].Amount)
		assert.Equal(t, 20.0, *payments[1].Amount)
		assert.Equal(t, 10.0, *payments[2].Amount)
	})

	t.Run("the payments are not found by the company", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.SavePayment(payment("company-1", "loc-1", 10, time.Now())))
		payments, err := repo.FindPayments("company-1")
		require.NoError(t, err)
		assert.Empty(t, payments)
	})

	t.Run("a location without payments has none", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		payments, err := repo.FindPayments("loc-1")
		require.NoError(t, err)
		assert.Empty(t, payments)
	})
}

func testSavePayment(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates and updates a payment", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		now := time.Now().Truncate(time.Second)
		p := payment("company-1", "loc-1", 10, now)
		p.StripeId = ldutils.Ptr("pi_1")
		require.NoError(t, repo.SavePayment(p))
		assert.NotZero(t, p.ID)

		amount := 15.5
		p.Amount = &amount
		require.NoError(t, repo.SavePayment(p))

		payments, err := repo.FindPayments("loc-1")
		require.NoError(t, err)
		require.Len(t, payments, 1)
		assert.Equal(t, p.ID, payments[0].ID)
		assert.Equal(t, 15.5, *payments[0].Amount)
		assert.Equal(t, "pi_1", *payments[0].StripeId)
		require.NotNil(t, payments[0].DateTime)
		assert.True(t, now.Equal(*payments[0].DateTime))
	})
}

func testFindPaidSubscription(t *testing.T, d ldconn.TestDriver) {
	t.Run("returns the subscription of the location", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.SavePaidSubscription(&ldspec.DbPaidSubscription{BusinessId: "company-1", LocationId: "loc-1", Status: ldspec.LDSubscriptionActive}))
		require.NoError(t, repo.SavePaidSubscription(&ldspec.DbPaidSubscription{BusinessId: "company-1", LocationId: "loc-2", Status: ldspec.LDSubscriptionCanceled}))

		subs, err := repo.FindPaidSubscription("loc-1")
		require.NoError(t, err)
		assert.Equal(t, "loc-1", subs.LocationId)
		assert.Equal(t, ldspec.LDSubscriptionActive, subs.Status)
	})

	t.Run("a deleted subscription is not found", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		sub := &ldspec.DbPaidSubscription{BusinessId: "company-1", LocationId: "loc-1", Status: ldspec.LDSubscriptionActive}
		require.NoError(t, repo.SavePaidSubscription(sub))
		require.NoError(t, repo.Db().Delete(sub).Error)

		subs, err := repo.FindPaidSubscription("loc-1")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, subs)
	})

	t.Run("a location without subscription has none", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		subs, err := repo.FindPaidSubscription("loc-1")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, subs)
	})
}

func testSavePaidSubscription(t *testing.T, d ldconn.TestDriver) {
	findAll := func(t *testing.T, repo ldcore.TestRepo) []*ldspec.DbPaidSubscription {
		t.Helper()
		var subs []*ldspec.DbPaidSubscription
		require.NoError(t, repo.Db().Order("id").Find(&subs).Error)
		return subs
	}

	t.Run("a new record updates the subscription of the same location", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first := &ldspec.DbPaidSubscription{
			BusinessId:    "company-1",
			LocationId:    "loc-1",
			Status:        ldspec.LDSubscriptionActive,
			MonthlyAmount: ldutils.Ptr(99.0),
			PerUserAmount: ldutils.Ptr(10.0),
			AmountDue:     ldutils.Ptr(0.0),
			Accounts:      ldutils.Ptr(uint(2)),
		}
		require.NoError(t, repo.SavePaidSubscription(first))
		assert.NotZero(t, first.ID)

		lastPayment := time.Now().Truncate(time.Second)
		second := &ldspec.DbPaidSubscription{
			BusinessId:       "company-2",
			LocationId:       "loc-1",
			StripeCustomerId: ldutils.Ptr("sub_1"),
			Status:           ldspec.LDSubscriptionUnPaid,
			MonthlyAmount:    ldutils.Ptr(149.0),
			PerUserAmount:    ldutils.Ptr(12.5),
			AmountDue:        ldutils.Ptr(74.5),
			Accounts:         ldutils.Ptr(uint(5)),
			LastPayment:      &lastPayment,
		}
		require.NoError(t, repo.SavePaidSubscription(second))
		assert.Equal(t, first.ID, second.ID)

		subs := findAll(t, repo)
		require.Len(t, subs, 1)
		assert.Equal(t, first.ID, subs[0].ID)
		assert.Equal(t, "company-2", subs[0].BusinessId)
		assert.Equal(t, "sub_1", *subs[0].StripeCustomerId)
		assert.Equal(t, ldspec.LDSubscriptionUnPaid, subs[0].Status)
		assert.Equal(t, 149.0, *subs[0].MonthlyAmount)
		assert.Equal(t, 12.5, *subs[0].PerUserAmount)
		assert.Equal(t, 74.5, *subs[0].AmountDue)
		assert.EqualValues(t, 5, *subs[0].Accounts)
		require.NotNil(t, subs[0].LastPayment)
		assert.True(t, lastPayment.Equal(*subs[0].LastPayment))
	})

	t.Run("a deleted subscription is restored", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first := &ldspec.DbPaidSubscription{BusinessId: "company-1", LocationId: "loc-1", Status: ldspec.LDSubscriptionActive}
		require.NoError(t, repo.SavePaidSubscription(first))
		require.NoError(t, repo.Db().Delete(first).Error)
		assert.Empty(t, findAll(t, repo))

		require.NoError(t, repo.SavePaidSubscription(&ldspec.DbPaidSubscription{BusinessId: "company-1", LocationId: "loc-1", Status: ldspec.LDSubscriptionCanceled}))
		subs := findAll(t, repo)
		require.Len(t, subs, 1)
		assert.Equal(t, first.ID, subs[0].ID)
		assert.Equal(t, ldspec.LDSubscriptionCanceled, subs[0].Status)
	})

	t.Run("a record with an id is saved as is", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		sub := &ldspec.DbPaidSubscription{BusinessId: "company-1", LocationId: "loc-1", Status: ldspec.LDSubscriptionActive}
		require.NoError(t, repo.SavePaidSubscription(sub))

		sub.Status = ldspec.LDSubscriptionCanceled
		require.NoError(t, repo.SavePaidSubscription(sub))

		subs := findAll(t, repo)
		require.Len(t, subs, 1)
		assert.Equal(t, ldspec.LDSubscriptionCanceled, subs[0].Status)
	})

	t.Run("different locations are kept apart", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		require.NoError(t, repo.SavePaidSubscription(&ldspec.DbPaidSubscription{BusinessId: "company-1", LocationId: "loc-1"}))
		require.NoError(t, repo.SavePaidSubscription(&ldspec.DbPaidSubscription{BusinessId: "company-1", LocationId: "loc-2"}))
		assert.Len(t, findAll(t, repo), 2)
	})
}

func testConcurrentSaves(t *testing.T, d ldconn.TestDriver) {
	d.SkipConcurrentWrites(t)

	t.Run("new records with the same id keep one record", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		var wg sync.WaitGroup
		errs := make(chan error, 20)
		for i := 0; i < 10; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				errs <- repo.SaveNode(&ldspec.DbServerNode{NodeId: "node-1", Status: ldspec.Live})
			}()
			go func() {
				defer wg.Done()
				errs <- repo.SaveLocation(location("company-1", "loc-1", "code-1"))
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			assert.NoError(t, err)
		}

		var nodes, locations int64
		require.NoError(t, repo.Db().Model(&ldspec.DbServerNode{}).Count(&nodes).Error)
		require.NoError(t, repo.Db().Model(&ldspec.DbGHLLocation{}).Count(&locations).Error)
		assert.EqualValues(t, 1, nodes)
		assert.EqualValues(t, 1, locations)
	})

	t.Run("a new record restores the deleted one with the same id", func(t *testing.T) {
		repo := ldcore.NewTestRepo(t, d)
		first := &ldspec.DbServerNode{NodeId: "node-1", Status: ldspec.Live}
		require.NoError(t, repo.SaveNode(first))
		require.NoError(t, repo.Db().Delete(first).Error)

		second := &ldspec.DbServerNode{NodeId: "node-1", Status: ldspec.Starting}
		require.NoError(t, repo.SaveNode(second))

		found, err := repo.FindNode("node-1")
		require.NoError(t, err)
		assert.Equal(t, first.ID, found.ID)
		assert.Equal(t, ldspec.Starting, found.Status)
	})
}
