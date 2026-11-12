package ldtenant_test

import (
	ldconn "ld-backend/internal/impl/back/db/conn"
	ldtenant "ld-backend/internal/impl/back/db/tenant"
	ldutils "ld-backend/internal/impl/core/utils"
	ldspec "ld-backend/internal/spec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// saveDialerSession saves a dialer session of the user in the node.
func saveDialerSession(t *testing.T, repo ldspec.TenantRepo, sessionId string, nodeId string, user *ldspec.DbUser, fill func(session *ldspec.DbDialerSession)) *ldspec.DbDialerSession {
	t.Helper()
	session := &ldspec.DbDialerSession{SessionId: sessionId, NodeId: nodeId, LocationId: "loc-1", CompanyId: "company-1"}
	if user != nil {
		session.UserId = &user.ID
	}
	if fill != nil {
		fill(session)
	}
	require.NoError(t, repo.SaveDialerSession(session))
	return session
}

func sessionIds(sessions []*ldspec.DbDialerSession) []string {
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.SessionId)
	}
	return ids
}

func testSaveDialerSession(t *testing.T, d ldconn.TestDriver) {
	t.Run("creates and updates a session", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		user := saveUser(t, repo, "user-1", "User 1")
		session := saveDialerSession(t, repo, "session-1", "node-1", user, nil)
		require.NotZero(t, session.ID)

		session.CallStatus = ldutils.Ptr(ldspec.CallStatusTalking)
		session.Tags = "hot,new"
		require.NoError(t, repo.SaveDialerSession(session))

		found, err := repo.FindDialerSessionBySessionId("session-1")
		require.NoError(t, err)
		assert.Equal(t, session.ID, found.ID)
		assert.Equal(t, ldspec.CallStatusTalking, *found.CallStatus)
		assert.Equal(t, "hot,new", found.Tags)
	})

	t.Run("unknown session id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		_, err := repo.FindDialerSessionBySessionId("unknown")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("duplicated session id", func(t *testing.T) {
		repo := ldtenant.NewTestRepo(t, d)
		saveDialerSession(t, repo, "session-1", "node-1", nil, nil)
		assert.Error(t, repo.SaveDialerSession(&ldspec.DbDialerSession{SessionId: "session-1"}))
	})
}

func testFindDialerSessions(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	user := saveUser(t, repo, "user-1", "User 1")
	other := saveUser(t, repo, "user-2", "User 2")
	pipeline := savePipeline(t, repo, "pipeline-1")
	contact := saveContact(t, repo, "contact-1", "+15550000001")
	did := saveDid(t, repo, "15550000002", true)
	call := saveCall(t, repo, "call-1", time.Now(), func(call *ldspec.DbCall) {
		call.ContactId = &contact.ID
		call.DidId = &did.ID
	})
	active := ldutils.Ptr(ldspec.CallerStatusActive)
	first := saveDialerSession(t, repo, "session-1", "node-1", user, func(session *ldspec.DbDialerSession) {
		session.CallerStatus = active
		session.PipelineId = &pipeline.ID
		session.LastCallId = &call.ID
	})
	// the updated times must differ to be ordered
	time.Sleep(10 * time.Millisecond)
	saveDialerSession(t, repo, "session-2", "node-1", user, nil)
	time.Sleep(10 * time.Millisecond)
	saveDialerSession(t, repo, "session-other", "node-1", other, func(session *ldspec.DbDialerSession) {
		session.CallerStatus = active
	})
	saveDialerSession(t, repo, "session-other-location", "node-1", other, func(session *ldspec.DbDialerSession) {
		session.CallerStatus = active
		session.LocationId = "loc-2"
	})
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, repo.SaveDialerSession(first))

	t.Run("sessions of the user, the latest updated first", func(t *testing.T) {
		sessions, err := repo.FindDialerSessions(&user.ID)
		require.NoError(t, err)
		require.Equal(t, []string{"session-1", "session-2"}, sessionIds(sessions))
		session := sessions[0]
		require.NotNil(t, session.User)
		assert.Equal(t, "user-1", *session.User.GHLId)
		require.NotNil(t, session.Pipeline)
		assert.Equal(t, "pipeline-1", session.Pipeline.GHLId)
		require.NotNil(t, session.LastCall)
		require.NotNil(t, session.LastCall.Contact)
		assert.Equal(t, "contact-1", session.LastCall.Contact.GHLId)
		require.NotNil(t, session.LastCall.Did)
		assert.Equal(t, "15550000002", session.LastCall.Did.DidNumber)
	})

	t.Run("user without sessions", func(t *testing.T) {
		sessions, err := repo.FindDialerSessions(ldutils.Ptr(uint(999)))
		require.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("open sessions of the location, the latest updated first", func(t *testing.T) {
		sessions, err := repo.FindOpenDialerSessions("loc-1")
		require.NoError(t, err)
		assert.Equal(t, []string{"session-1", "session-other"}, sessionIds(sessions))
	})

	t.Run("location without open sessions", func(t *testing.T) {
		sessions, err := repo.FindOpenDialerSessions("loc-3")
		require.NoError(t, err)
		assert.Empty(t, sessions)
	})
}

func testRemoveOldDialerSessions(t *testing.T, d ldconn.TestDriver) {
	repo := ldtenant.NewTestRepo(t, d)
	saveDialerSession(t, repo, "session-node-1", "node-1", nil, nil)
	saveDialerSession(t, repo, "session-node-2", "node-2", nil, nil)
	saveDialerSession(t, repo, "session-no-node", "", nil, nil)

	require.NoError(t, repo.RemoveOldDialerSessions("node-1"))

	t.Run("permanently deletes the sessions of the node and without a node", func(t *testing.T) {
		for _, sessionId := range []string{"session-node-1", "session-no-node"} {
			_, err := repo.FindDialerSessionBySessionId(sessionId)
			assert.ErrorIs(t, err, gorm.ErrRecordNotFound, sessionId)
		}
		var count int64
		require.NoError(t, repo.Db().Unscoped().Model(&ldspec.DbDialerSession{}).Count(&count).Error)
		assert.EqualValues(t, 1, count, "the sessions are not soft deleted")
	})

	t.Run("keeps the sessions of the other nodes", func(t *testing.T) {
		_, err := repo.FindDialerSessionBySessionId("session-node-2")
		assert.NoError(t, err)
	})
}
