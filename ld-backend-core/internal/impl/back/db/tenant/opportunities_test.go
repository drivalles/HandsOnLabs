package ldtenant_test

import (
	ldconn "ld-backend/internal/impl/back/db/conn"
	ldtenant "ld-backend/internal/impl/back/db/tenant"
	ldspec "ld-backend/internal/spec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// saveOpportunity saves an opportunity of the contact in the stage.
func saveOpportunity(t *testing.T, repo ldspec.TenantRepo, ghlId string, contact *ldspec.DbContact, stage *ldspec.DbPipelineStage, assignedTo *ldspec.DbUser) *ldspec.DbOpportunity {
	t.Helper()
	opportunity := &ldspec.DbOpportunity{GHLId: ghlId, ContactId: contact.ID, PipelineId: stage.PipelineId, StageId: stage.ID}
	if assignedTo != nil {
		opportunity.AssignedToId = &assignedTo.ID
	}
	require.NoError(t, repo.SaveOpportunity(opportunity))
	return opportunity
}

func opportunityGHLIds(opportunities []*ldspec.DbOpportunity) []string {
	ids := make([]string, 0, len(opportunities))
	for _, opportunity := range opportunities {
		ids = append(ids, opportunity.GHLId)
	}
	return ids
}

func testSaveOpportunity(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates and updates an opportunity", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		first := saveStage(t, repo, pipeline, "stage-1")
		second := saveStage(t, repo, pipeline, "stage-2")
		contact := saveContact(t, repo, "contact-1", "+15550000001")
		opportunity := saveOpportunity(t, repo, "opp-1", contact, first, nil)
		require.NotZero(t, opportunity.ID)

		opportunity.StageId = second.ID
		require.NoError(t, repo.SaveOpportunity(opportunity))

		found, err := repo.FindOpportunityByGHLId("opp-1")
		require.NoError(t, err)
		assert.Equal(t, opportunity.ID, found.ID)
		assert.Equal(t, second.ID, found.StageId)
	})

	t.Run("permanently deletes an opportunity", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		stage := saveStage(t, repo, pipeline, "stage-1")
		contact := saveContact(t, repo, "contact-1", "+15550000001")
		opportunity := saveOpportunity(t, repo, "opp-1", contact, stage, nil)
		saveOpportunity(t, repo, "opp-2", contact, stage, nil)

		require.NoError(t, repo.DeleteOpportunity(opportunity))

		_, err := repo.FindOpportunityByGHLId("opp-1")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		var count int64
		require.NoError(t, repo.Db().Unscoped().Model(&ldspec.DbOpportunity{}).Count(&count).Error)
		assert.EqualValues(t, 1, count, "the opportunity is not soft deleted")
	})
}

func testFindOpportunities(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	user := saveUser(t, repo, "user-1", "User 1")
	pipeline := savePipeline(t, repo, "pipeline-1")
	otherPipeline := savePipeline(t, repo, "pipeline-2")
	stage := saveStage(t, repo, pipeline, "stage-1")
	otherStage := saveStage(t, repo, pipeline, "stage-2")
	otherPipelineStage := saveStage(t, repo, otherPipeline, "stage-3")
	contact := saveContact(t, repo, "contact-1", "+15550000001")
	otherContact := saveContact(t, repo, "contact-2", "+15550000002")
	saveOpportunity(t, repo, "opp-assigned", contact, stage, user)
	saveOpportunity(t, repo, "opp-unassigned", otherContact, stage, nil)
	saveOpportunity(t, repo, "opp-other-stage", contact, otherStage, user)
	saveOpportunity(t, repo, "opp-other-pipeline", otherContact, otherPipelineStage, user)

	t.Run("opportunities in the stage of the pipeline", func(t *testing.T) {
		opportunities, err := repo.FindOpportunities(pipeline, stage, nil)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"opp-assigned", "opp-unassigned"}, opportunityGHLIds(opportunities))
	})

	t.Run("only those assigned to the user", func(t *testing.T) {
		opportunities, err := repo.FindOpportunities(pipeline, stage, user)
		require.NoError(t, err)
		assert.Equal(t, []string{"opp-assigned"}, opportunityGHLIds(opportunities))
	})

	t.Run("opportunities of the pipeline", func(t *testing.T) {
		opportunities, err := repo.FindOpportunitiesByPipeline(pipeline)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"opp-assigned", "opp-unassigned", "opp-other-stage"}, opportunityGHLIds(opportunities))
	})

	t.Run("opportunities of the contact", func(t *testing.T) {
		opportunities, err := repo.FindOpportunitiesByContact(contact)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"opp-assigned", "opp-other-stage"}, opportunityGHLIds(opportunities))
	})

	t.Run("opportunity of the contact in the pipeline", func(t *testing.T) {
		opportunity, err := repo.FindOpportunity(otherPipeline, otherContact)
		require.NoError(t, err)
		assert.Equal(t, "opp-other-pipeline", opportunity.GHLId)

		_, err = repo.FindOpportunity(otherPipeline, contact)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("finds by GHL id with its relations", func(t *testing.T) {
		opportunity, err := repo.FindOpportunityByGHLId("opp-assigned")
		require.NoError(t, err)
		require.NotNil(t, opportunity.Contact)
		assert.Equal(t, "contact-1", opportunity.Contact.GHLId)
		require.NotNil(t, opportunity.Pipeline)
		assert.Equal(t, "pipeline-1", opportunity.Pipeline.GHLId)
		require.NotNil(t, opportunity.Stage)
		assert.Equal(t, "stage-1", opportunity.Stage.GHLId)
		require.NotNil(t, opportunity.AssignedTo)
		assert.Equal(t, "user-1", *opportunity.AssignedTo.GHLId)
	})

	t.Run("unknown GHL id", func(t *testing.T) {
		_, err := repo.FindOpportunityByGHLId("unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func testFindNewOpportunities(t *testing.T, d ldconn.TestDriver) {
	t.Run("opportunities in the new leads stage of the contacts registered in the last day", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		leadsStage := saveStage(t, repo, pipeline, "stage-leads")
		otherStage := saveStage(t, repo, pipeline, "stage-other")
		require.NoError(t, repo.SetNewLeadsStage(leadsStage.ID))
		now := time.Now()
		old := now.Add(-3 * 24 * time.Hour)
		newContact := &ldspec.DbContact{GHLId: "contact-new", DateRegistered: &now}
		oldContact := &ldspec.DbContact{GHLId: "contact-old", DateRegistered: &old}
		require.NoError(t, repo.SaveContact(newContact))
		require.NoError(t, repo.SaveContact(oldContact))
		saveOpportunity(t, repo, "opp-new", newContact, leadsStage, nil)
		saveOpportunity(t, repo, "opp-old", oldContact, leadsStage, nil)
		saveOpportunity(t, repo, "opp-other-stage", newContact, otherStage, nil)

		opportunities, err := repo.FindNewOpportunities()
		require.NoError(t, err)
		assert.Equal(t, []string{"opp-new"}, opportunityGHLIds(opportunities))
	})

	t.Run("none without the new leads stage", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		pipeline := savePipeline(t, repo, "pipeline-1")
		stage := saveStage(t, repo, pipeline, "stage-1")
		now := time.Now()
		contact := &ldspec.DbContact{GHLId: "contact-new", DateRegistered: &now}
		require.NoError(t, repo.SaveContact(contact))
		saveOpportunity(t, repo, "opp-new", contact, stage, nil)

		opportunities, err := repo.FindNewOpportunities()
		require.NoError(t, err)
		assert.NotNil(t, opportunities)
		assert.Empty(t, opportunities)
	})
}

func testFindCallbackOpportunities(t *testing.T, d ldconn.TestDriver) {
	t.Run("no due callbacks", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		opportunities, err := repo.FindCallbackOpportunities(nil)
		require.NoError(t, err)
		assert.NotNil(t, opportunities)
		assert.Empty(t, opportunities)
	})

	repo := ldtenant.NewTestRepo(t, d)
	user := saveUser(t, repo, "user-1", "User 1")
	other := saveUser(t, repo, "user-2", "User 2")
	pipeline := savePipeline(t, repo, "pipeline-1")
	stage := saveStage(t, repo, pipeline, "stage-1")
	for _, callback := range []struct {
		name string
		user *ldspec.DbUser
		at   time.Time
	}{
		{"due-user", user, time.Now().Add(-time.Hour)},
		{"due-other", other, time.Now().Add(-time.Minute)},
		{"later", user, time.Now().Add(time.Hour)},
	} {
		contact := saveContact(t, repo, "contact-"+callback.name, "+15550000001")
		require.NoError(t, repo.UpdateContactCallback(contact.GHLId, callback.user.ID, callback.at))
		saveOpportunity(t, repo, "opp-"+callback.name, contact, stage, nil)
	}
	saveOpportunity(t, repo, "opp-no-callback", saveContact(t, repo, "contact-none", "+15550000002"), stage, nil)

	t.Run("opportunities of the contacts with a due callback", func(t *testing.T) {
		opportunities, err := repo.FindCallbackOpportunities(nil)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"opp-due-user", "opp-due-other"}, opportunityGHLIds(opportunities))
	})

	t.Run("only the callbacks of the user, with the user", func(t *testing.T) {
		opportunities, err := repo.FindCallbackOpportunities(user)
		require.NoError(t, err)
		require.Equal(t, []string{"opp-due-user"}, opportunityGHLIds(opportunities))
		require.NotNil(t, opportunities[0].Contact)
		require.NotNil(t, opportunities[0].Contact.CallbackUser)
		assert.Equal(t, "user-1", *opportunities[0].Contact.CallbackUser.GHLId)
	})
}
