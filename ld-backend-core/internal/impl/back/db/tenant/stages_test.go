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

func stageGHLIds(stages []*ldspec.DbPipelineStage) []string {
	ids := make([]string, 0, len(stages))
	for _, stage := range stages {
		ids = append(ids, stage.GHLId)
	}
	return ids
}

func testSaveStage(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates and updates a stage", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		stage := saveStage(t, repo, pipeline, "stage-1")
		require.NotZero(t, stage.ID)

		stage.Name = "Renamed"
		require.NoError(t, repo.SaveStage(stage))

		found, err := repo.FindStageById(stage.ID)
		require.NoError(t, err)
		assert.Equal(t, "Renamed", found.Name)
	})

	t.Run("duplicated GHL id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		saveStage(t, repo, pipeline, "stage-1")
		assert.Error(t, repo.SaveStage(&ldspec.DbPipelineStage{PipelineId: pipeline.ID, GHLId: "stage-1"}))
	})

	t.Run("permanently deletes a stage", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		stage := saveStage(t, repo, pipeline, "stage-1")
		saveStage(t, repo, pipeline, "stage-2")

		require.NoError(t, repo.DeleteStage(stage))

		_, err := repo.FindStageById(stage.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		var count int64
		require.NoError(t, repo.Db().Unscoped().Model(&ldspec.DbPipelineStage{}).Count(&count).Error)
		assert.EqualValues(t, 1, count, "the stage is not soft deleted")
	})
}

func testFindStages(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	pipeline := savePipeline(t, repo, "pipeline-1")
	other := savePipeline(t, repo, "pipeline-2")
	saveStage(t, repo, pipeline, "stage-1")
	stage := saveStage(t, repo, pipeline, "stage-2")
	saveStage(t, repo, other, "stage-3")
	empty := savePipeline(t, repo, "pipeline-3")

	t.Run("finds by id with its pipeline", func(t *testing.T) {
		found, err := repo.FindStageById(stage.ID)
		require.NoError(t, err)
		assert.Equal(t, "stage-2", found.GHLId)
		require.NotNil(t, found.Pipeline)
		assert.Equal(t, "pipeline-1", found.Pipeline.GHLId)
	})

	t.Run("finds by GHL id in the pipeline", func(t *testing.T) {
		found, err := repo.FindStageByGHLId(pipeline.ID, "stage-2")
		require.NoError(t, err)
		assert.Equal(t, stage.ID, found.ID)

		_, err = repo.FindStageByGHLId(other.ID, "stage-2")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "a stage of another pipeline")
	})

	t.Run("stages of the pipeline id", func(t *testing.T) {
		stages, err := repo.FindStagesByPipelineId(pipeline.ID)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"stage-1", "stage-2"}, stageGHLIds(stages))
	})

	t.Run("stages of the pipeline with their pipeline", func(t *testing.T) {
		stages := repo.FindStagesByPipeline(pipeline)
		assert.ElementsMatch(t, []string{"stage-1", "stage-2"}, stageGHLIds(stages))
		for _, s := range stages {
			require.NotNil(t, s.Pipeline)
			assert.Equal(t, "pipeline-1", s.Pipeline.GHLId)
		}
	})

	t.Run("pipeline without stages", func(t *testing.T) {
		stages, err := repo.FindStagesByPipelineId(empty.ID)
		require.NoError(t, err)
		assert.Empty(t, stages)
		assert.Empty(t, repo.FindStagesByPipeline(empty))
	})

	t.Run("unknown ids", func(t *testing.T) {
		_, err := repo.FindStageById(999)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = repo.FindStageByGHLId(pipeline.ID, "unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func testDeleteStagesByPipelineId(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	pipeline := savePipeline(t, repo, "pipeline-1")
	other := savePipeline(t, repo, "pipeline-2")
	saveStage(t, repo, pipeline, "stage-1")
	saveStage(t, repo, pipeline, "stage-2")
	saveStage(t, repo, other, "stage-3")

	require.NoError(t, repo.DeleteStagesByPipelineId(pipeline.ID))

	t.Run("permanently deletes the stages of the pipeline", func(t *testing.T) {
		stages, err := repo.FindStagesByPipelineId(pipeline.ID)
		require.NoError(t, err)
		assert.Empty(t, stages)
		var count int64
		require.NoError(t, repo.Db().Unscoped().Model(&ldspec.DbPipelineStage{}).Where("pipeline_id = ?", pipeline.ID).Count(&count).Error)
		assert.Zero(t, count, "the stages are not soft deleted")
	})

	t.Run("keeps the stages of the other pipelines", func(t *testing.T) {
		stages, err := repo.FindStagesByPipelineId(other.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"stage-3"}, stageGHLIds(stages))
	})

	t.Run("pipeline without stages", func(t *testing.T) {
		assert.NoError(t, repo.DeleteStagesByPipelineId(999))
	})
}
