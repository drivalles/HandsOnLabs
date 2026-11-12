package ldtenant_test

import (
	ldconn "ld-backend/internal/impl/back/db/conn"
	ldtenant "ld-backend/internal/impl/back/db/tenant"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewLeadsStage(t *testing.T, d ldconn.TestDriver) {
	t.Run("not set", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		stage, err := repo.GetNewLeadsStage()
		require.NoError(t, err)
		assert.Nil(t, stage)
	})

	t.Run("sets the stage, returned with its pipeline", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		stage := saveStage(t, repo, pipeline, "stage-1")

		require.NoError(t, repo.SetNewLeadsStage(stage.ID))

		found, err := repo.GetNewLeadsStage()
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, stage.ID, found.ID)
		require.NotNil(t, found.Pipeline)
		assert.Equal(t, "pipeline-1", found.Pipeline.GHLId)
	})

	t.Run("changes the stage", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		first := saveStage(t, repo, pipeline, "stage-1")
		second := saveStage(t, repo, pipeline, "stage-2")
		require.NoError(t, repo.SetNewLeadsStage(first.ID))

		require.NoError(t, repo.SetNewLeadsStage(second.ID))

		found, err := repo.GetNewLeadsStage()
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, second.ID, found.ID)
	})

	t.Run("keeps the disposition field", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		stage := saveStage(t, repo, pipeline, "stage-1")
		require.NoError(t, repo.SetDispositionField("field-1"))

		require.NoError(t, repo.SetNewLeadsStage(stage.ID))

		field, err := repo.GetDispositionField()
		require.NoError(t, err)
		assert.Equal(t, "field-1", field)
	})
}

func testDispositionField(t *testing.T, d ldconn.TestDriver) {
	t.Run("not set", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		field, err := repo.GetDispositionField()
		require.NoError(t, err)
		assert.Empty(t, field)
	})

	t.Run("sets and changes the field", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		require.NoError(t, repo.SetDispositionField("field-1"))
		field, err := repo.GetDispositionField()
		require.NoError(t, err)
		assert.Equal(t, "field-1", field)

		require.NoError(t, repo.SetDispositionField("field-2"))
		field, err = repo.GetDispositionField()
		require.NoError(t, err)
		assert.Equal(t, "field-2", field)
	})

	t.Run("keeps the new leads stage", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		stage := saveStage(t, repo, pipeline, "stage-1")
		require.NoError(t, repo.SetNewLeadsStage(stage.ID))

		require.NoError(t, repo.SetDispositionField("field-1"))

		found, err := repo.GetNewLeadsStage()
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, stage.ID, found.ID)
	})
}
