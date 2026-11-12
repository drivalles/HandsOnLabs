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

func testSavePipeline(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates and updates a pipeline", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		require.NotZero(t, pipeline.ID)

		pipeline.Name = "Renamed"
		require.NoError(t, repo.SavePipeline(pipeline))

		found, err := repo.FindPipeline("pipeline-1")
		require.NoError(t, err)
		assert.Equal(t, pipeline.ID, found.ID)
		assert.Equal(t, "Renamed", found.Name)
	})

	t.Run("duplicated GHL id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		savePipeline(t, repo, "pipeline-1")
		assert.Error(t, repo.SavePipeline(&ldspec.DbPipeline{GHLId: "pipeline-1"}))
	})

	t.Run("permanently deletes a pipeline", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		savePipeline(t, repo, "pipeline-2")

		require.NoError(t, repo.DeletePipeline(pipeline))

		_, err := repo.FindPipeline("pipeline-1")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		var count int64
		require.NoError(t, repo.Db().Unscoped().Model(&ldspec.DbPipeline{}).Count(&count).Error)
		assert.EqualValues(t, 1, count, "the pipeline is not soft deleted")
	})
}

func testFindPipelines(t *testing.T, d ldconn.TestDriver) {
	t.Run("no pipelines", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipelines, err := repo.FindPipelines()
		require.NoError(t, err)
		assert.Empty(t, pipelines)
	})

	repo := ldtenant.NewTestRepo(t, d)
	savePipeline(t, repo, "pipeline-1")
	pipeline := savePipeline(t, repo, "pipeline-2")

	t.Run("all the pipelines", func(t *testing.T) {
		pipelines, err := repo.FindPipelines()
		require.NoError(t, err)
		ids := []string{}
		for _, p := range pipelines {
			ids = append(ids, p.GHLId)
		}
		assert.ElementsMatch(t, []string{"pipeline-1", "pipeline-2"}, ids)
	})

	t.Run("finds by GHL id", func(t *testing.T) {
		found, err := repo.FindPipeline("pipeline-2")
		require.NoError(t, err)
		assert.Equal(t, pipeline.ID, found.ID)
	})

	t.Run("finds by id", func(t *testing.T) {
		found, err := repo.FindPipelineById(pipeline.ID)
		require.NoError(t, err)
		assert.Equal(t, "pipeline-2", found.GHLId)
	})

	t.Run("unknown ids", func(t *testing.T) {
		_, err := repo.FindPipeline("unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = repo.FindPipelineById(999)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
