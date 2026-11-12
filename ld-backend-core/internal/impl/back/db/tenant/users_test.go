package ldtenant_test

import (
	ldconn "ld-backend/internal/impl/back/db/conn"
	ldtenant "ld-backend/internal/impl/back/db/tenant"
	ldutils "ld-backend/internal/impl/core/utils"
	ldspec "ld-backend/internal/spec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testFindOrCreateUser(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates a missing user with the name", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user, err := repo.FindOrCreateUser(ldutils.Ptr("user-1"), "User 1")
		require.NoError(t, err)
		assert.NotZero(t, user.ID)
		assert.Equal(t, "User 1", *user.UserName)

		found, err := repo.FindUser(ldutils.Ptr("user-1"))
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
	})

	t.Run("finds an existing user keeping its name", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		created := saveUser(t, repo, "user-1", "User 1")

		user, err := repo.FindOrCreateUser(ldutils.Ptr("user-1"), "Other name")
		require.NoError(t, err)
		assert.Equal(t, created.ID, user.ID)
		assert.Equal(t, "User 1", *user.UserName)
		users, err := repo.FindUsers()
		require.NoError(t, err)
		assert.Len(t, users, 1)
	})

	t.Run("sets the name of an existing user without one", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		created := &ldspec.DbUser{GHLId: ldutils.Ptr("user-1")}
		require.NoError(t, repo.SaveUser(created))

		user, err := repo.FindOrCreateUser(ldutils.Ptr("user-1"), "User 1")
		require.NoError(t, err)
		assert.Equal(t, created.ID, user.ID)
		found, err := repo.FindUser(ldutils.Ptr("user-1"))
		require.NoError(t, err)
		assert.Equal(t, "User 1", *found.UserName)
	})

	t.Run("an empty name keeps the name", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		saveUser(t, repo, "user-1", "User 1")
		user, err := repo.FindOrCreateUser(ldutils.Ptr("user-1"), "")
		require.NoError(t, err)
		assert.Equal(t, "User 1", *user.UserName)
	})

	t.Run("a conflicting transaction is tried again", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		creates := ldconn.ConflictOnCreate(t, repo.Db(), "users", 1)

		user, err := repo.FindOrCreateUser(ldutils.Ptr("user-1"), "User 1")
		require.NoError(t, err)
		assert.EqualValues(t, 2, creates.Load())
		users, err := repo.FindUsers()
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, user.ID, users[0].ID)
	})

	t.Run("nil id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user, err := repo.FindOrCreateUser(nil, "User 1")
		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func testFindUsers(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	saveUser(t, repo, "user-1", "User 1")
	user := saveUser(t, repo, "user-2", "User 2")

	t.Run("all the users", func(t *testing.T) {
		users, err := repo.FindUsers()
		require.NoError(t, err)
		assert.Len(t, users, 2)
	})

	t.Run("finds by GHL id", func(t *testing.T) {
		found, err := repo.FindUser(ldutils.Ptr("user-2"))
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
	})

	t.Run("updates a user", func(t *testing.T) {
		user.Email = ldutils.Ptr("user2@example.com")
		require.NoError(t, repo.SaveUser(user))
		found, err := repo.FindUser(ldutils.Ptr("user-2"))
		require.NoError(t, err)
		assert.Equal(t, "user2@example.com", *found.Email)
	})

	t.Run("unknown GHL id", func(t *testing.T) {
		_, err := repo.FindUser(ldutils.Ptr("unknown"))
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("nil id", func(t *testing.T) {
		user, err := repo.FindUser(nil)
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	t.Run("duplicated GHL id", func(t *testing.T) {
		assert.Error(t, repo.SaveUser(&ldspec.DbUser{GHLId: ldutils.Ptr("user-1")}))
	})
}

func testUpdateUserAccess(t *testing.T, d ldconn.TestDriver) {
	t.Run("sets the access of the user", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		saveUser(t, repo, "user-2", "User 2")

		require.NoError(t, repo.UpdateUserAccess(user.ID, ldspec.LDUserAccessSupervisor))

		found, err := repo.FindUser(ldutils.Ptr("user-1"))
		require.NoError(t, err)
		require.NotNil(t, found.LDAccessType)
		assert.Equal(t, ldspec.LDUserAccessSupervisor, *found.LDAccessType)
		assert.Equal(t, "User 1", *found.UserName, "the other fields are kept")
		found, err = repo.FindUser(ldutils.Ptr("user-2"))
		require.NoError(t, err)
		assert.Nil(t, found.LDAccessType, "the other users are kept")
	})

	t.Run("changes the access", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		require.NoError(t, repo.UpdateUserAccess(user.ID, ldspec.LDUserAccessDialer))
		require.NoError(t, repo.UpdateUserAccess(user.ID, ldspec.LDUserAccessOwner))

		found, err := repo.FindUser(ldutils.Ptr("user-1"))
		require.NoError(t, err)
		assert.Equal(t, ldspec.LDUserAccessOwner, *found.LDAccessType)
	})

	t.Run("unknown user", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		err := repo.UpdateUserAccess(999, ldspec.LDUserAccessDialer)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
