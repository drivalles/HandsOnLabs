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

func testSaveDisposition(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates a disposition", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		disposition := &ldspec.DbDisposition{Name: "Interested", GHLId: ldutils.Ptr("option-1")}
		require.NoError(t, repo.SaveDisposition(disposition))
		assert.NotZero(t, disposition.ID)

		found, err := repo.FindDispositionById(disposition.ID)
		require.NoError(t, err)
		assert.Equal(t, "Interested", found.Name)
		assert.Equal(t, "option-1", *found.GHLId)
	})

	t.Run("updates the name of a disposition by id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		disposition := &ldspec.DbDisposition{Name: "Interested"}
		require.NoError(t, repo.SaveDisposition(disposition))

		require.NoError(t, repo.SaveDisposition(&ldspec.DbDisposition{Model: gorm.Model{ID: disposition.ID}, Name: "Very interested"}))

		found, err := repo.FindDispositionById(disposition.ID)
		require.NoError(t, err)
		assert.Equal(t, "Very interested", found.Name)
	})

	t.Run("a new record updates the disposition with the same GHL id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		disposition := &ldspec.DbDisposition{Name: "Interested", GHLId: ldutils.Ptr("option-1")}
		require.NoError(t, repo.SaveDisposition(disposition))

		require.NoError(t, repo.SaveDisposition(&ldspec.DbDisposition{Name: "Renamed", GHLId: ldutils.Ptr("option-1")}))

		dispositions, err := repo.FindDispositions()
		require.NoError(t, err)
		require.Len(t, dispositions, 1)
		assert.Equal(t, disposition.ID, dispositions[0].ID)
		assert.Equal(t, "Renamed", dispositions[0].Name)
	})

	t.Run("unknown id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		err := repo.SaveDisposition(&ldspec.DbDisposition{Model: gorm.Model{ID: 999}, Name: "Interested"})
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("a conflicting transaction creates the disposition again", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		creates := ldconn.ConflictOnCreate(t, repo.Db(), "dispositions", 1)
		disposition := &ldspec.DbDisposition{Name: "Interested", GHLId: ldutils.Ptr("option-1")}

		require.NoError(t, repo.SaveDisposition(disposition))

		assert.EqualValues(t, 2, creates.Load())
		dispositions, err := repo.FindDispositions()
		require.NoError(t, err)
		require.Len(t, dispositions, 1)
		assert.Equal(t, disposition.ID, dispositions[0].ID)
	})
}

func testFindDispositions(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	var interested *ldspec.DbDisposition
	for _, name := range []string{"Not interested", "Callback", "Interested"} {
		disposition := &ldspec.DbDisposition{Name: name}
		require.NoError(t, repo.SaveDisposition(disposition))
		if name == "Interested" {
			interested = disposition
		}
	}

	t.Run("all the dispositions by name", func(t *testing.T) {
		dispositions, err := repo.FindDispositions()
		require.NoError(t, err)
		names := []string{}
		for _, disposition := range dispositions {
			names = append(names, disposition.Name)
		}
		assert.Equal(t, []string{"Callback", "Interested", "Not interested"}, names)
	})

	t.Run("finds by id", func(t *testing.T) {
		found, err := repo.FindDispositionById(interested.ID)
		require.NoError(t, err)
		assert.Equal(t, "Interested", found.Name)
	})

	t.Run("unknown id", func(t *testing.T) {
		_, err := repo.FindDispositionById(999)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func testFindOrCreateDispositionByName(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates the disposition", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		disposition, err := repo.FindOrCreateDispositionByName("Callback")
		require.NoError(t, err)
		assert.NotZero(t, disposition.ID)
		assert.Equal(t, "Callback", disposition.Name)
	})

	t.Run("finds the existing disposition", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		existing := &ldspec.DbDisposition{Name: "Callback", GHLId: ldutils.Ptr("option-1")}
		require.NoError(t, repo.SaveDisposition(existing))

		disposition, err := repo.FindOrCreateDispositionByName("Callback")
		require.NoError(t, err)
		assert.Equal(t, existing.ID, disposition.ID)

		dispositions, err := repo.FindDispositions()
		require.NoError(t, err)
		assert.Len(t, dispositions, 1)
	})
}
