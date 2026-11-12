package ldtenant_test

import (
	ldconn "ld-backend/internal/impl/back/db/conn"
	ldtenant "ld-backend/internal/impl/back/db/tenant"
	ldspec "ld-backend/internal/spec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testSaveCallScript(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates a script", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		script := &ldspec.DbCallScript{Name: "Intro", Text: "Hello"}
		require.NoError(t, repo.SaveCallScript(script))
		assert.NotZero(t, script.ID)

		scripts, err := repo.FindCallScripts()
		require.NoError(t, err)
		require.Len(t, scripts, 1)
		assert.Equal(t, script.ID, scripts[0].ID)
		assert.Equal(t, "Hello", scripts[0].Text)
	})

	t.Run("updates the name and text of a script", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		script := &ldspec.DbCallScript{Name: "Intro", Text: "Hello"}
		require.NoError(t, repo.SaveCallScript(script))

		require.NoError(t, repo.SaveCallScript(&ldspec.DbCallScript{Model: gorm.Model{ID: script.ID}, Name: "Opening", Text: "Hi"}))

		scripts, err := repo.FindCallScripts()
		require.NoError(t, err)
		require.Len(t, scripts, 1)
		assert.Equal(t, "Opening", scripts[0].Name)
		assert.Equal(t, "Hi", scripts[0].Text)
	})

	t.Run("unknown id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		err := repo.SaveCallScript(&ldspec.DbCallScript{Model: gorm.Model{ID: 999}, Name: "Intro"})
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("a conflicting transaction creates the script again", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		creates := ldconn.ConflictOnCreate(t, repo.Db(), "call_scripts", 1)
		script := &ldspec.DbCallScript{Name: "Intro", Text: "Hello"}

		require.NoError(t, repo.SaveCallScript(script))

		assert.EqualValues(t, 2, creates.Load())
		scripts, err := repo.FindCallScripts()
		require.NoError(t, err)
		require.Len(t, scripts, 1)
		assert.Equal(t, script.ID, scripts[0].ID)
	})

	t.Run("gives up when the transaction keeps conflicting", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		ldconn.ConflictOnCreate(t, repo.Db(), "call_scripts", 100)

		err := repo.SaveCallScript(&ldspec.DbCallScript{Name: "Intro"})
		assert.True(t, ldconn.IsRetryable(err), "the conflict is returned: %v", err)
		scripts, err := repo.FindCallScripts()
		require.NoError(t, err)
		assert.Empty(t, scripts, "the script is rolled back")
	})

	t.Run("finds the scripts by name", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		for _, name := range []string{"Closing", "Intro", "Objections"} {
			require.NoError(t, repo.SaveCallScript(&ldspec.DbCallScript{Name: name}))
		}
		scripts, err := repo.FindCallScripts()
		require.NoError(t, err)
		names := []string{}
		for _, script := range scripts {
			names = append(names, script.Name)
		}
		assert.Equal(t, []string{"Closing", "Intro", "Objections"}, names)
	})
}
