/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Maxim Geraskin
 */

package invite

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

const (
	applyInviteEventsTestInviteEmail      = "InviteEmail"
	applyInviteEventsTestReplacementRoles = "app1pkg.SpecialAPITokenRole"
)

func newMockEventWithCUDs(t *testing.T, cmdQName appdef.QName, cuds []istructs.ICUDRow) *coreutils.MockPLogEvent {
	t.Helper()
	event := &coreutils.MockPLogEvent{}
	event.On("QName").Return(cmdQName)
	event.On("CUDs", mock.Anything).Run(func(args mock.Arguments) {
		cb := args.Get(0).(func(istructs.ICUDRow) bool)
		for _, c := range cuds {
			if !cb(c) {
				return
			}
		}
	})
	return event
}

func newInviteCUD(version int32, id istructs.RecordID) *coreutils.TestObject {
	return &coreutils.TestObject{
		Name: QNameCDocInvite,
		ID_:  id,
		Data: map[string]any{Field_Version: version},
	}
}

// TestApplyInviteEvents_SkipsEventsWithoutVersionMarker verifies that ap.sys.ApplyInviteEvents
// returns nil without touching state/intents/federation when the event's cdoc.sys.Invite CUD
// is missing or carries Version == 0 (pre-refactor events whose effects were already produced
// by deprecated per-command projectors). Strict mocks for IState and IIntents are used so that
// any unexpected call fails the test, making the "no side effects" half of the contract explicit.
func TestApplyInviteEvents_SkipsEventsWithoutVersionMarker(t *testing.T) {
	projectorFn := applyInviteEvents(nil, nil, nil, smtp.Cfg{})

	run := func(t *testing.T, cmdQName appdef.QName, cuds []istructs.ICUDRow) {
		t.Helper()
		require := require.New(t)
		st := &coreutils.MockState{}
		in := &coreutils.MockIntents{}
		event := newMockEventWithCUDs(t, cmdQName, cuds)
		require.NoError(projectorFn(event, st, in))
		st.AssertExpectations(t)
		in.AssertExpectations(t)
	}

	t.Run("no cdoc.sys.Invite CUD", func(t *testing.T) {
		run(t, qNameCmdInitiateInvitationByEMail, nil)
	})

	t.Run("cdoc.sys.Invite CUD with Version=0", func(t *testing.T) {
		run(t, qNameCmdInitiateJoinWorkspace, []istructs.ICUDRow{newInviteCUD(0, 42)})
	})

	t.Run("only non-invite CUDs", func(t *testing.T) {
		otherCUD := &coreutils.TestObject{
			Name: appdef.NewQName(appdef.SysPackage, "Subject"),
			ID_:  3,
			Data: map[string]any{},
		}
		run(t, qNameCmdCancelSentInvite, []istructs.ICUDRow{otherCUD})
	})

	t.Run("invite CUD listed after a non-invite CUD with Version=0", func(t *testing.T) {
		otherCUD := &coreutils.TestObject{
			Name: appdef.NewQName(appdef.SysPackage, "Subject"),
			Data: map[string]any{},
		}
		run(t, qNameCmdInitiateLeaveWorkspace, []istructs.ICUDRow{otherCUD, newInviteCUD(0, 7)})
	})
}

// TestApplyInviteEvents_Version1ReachesDispatch verifies that when the event's cdoc.sys.Invite
// CUD carries Version == 1 the projector progresses past the skip check and reaches loadInviteByID,
// proven by an injected state.KeyBuilder error propagating back as the projector's return value.
func TestApplyInviteEvents_Version1ReachesDispatch(t *testing.T) {
	require := require.New(t)
	expectedErr := errors.New("expected propagated error")

	st := &coreutils.MockState{}
	st.On("KeyBuilder", sys.Storage_Record, QNameCDocInvite).
		Return(&coreutils.MockStateKeyBuilder{}, expectedErr)

	projectorFn := applyInviteEvents(nil, nil, nil, smtp.Cfg{})

	// qNameCmdInitiateLeaveWorkspace: inviteIDFromEvent reads InviteID directly from the located
	// CUD (no state interaction), so the next state call is loadInviteByID -> s.KeyBuilder, whose
	// error propagation proves the projector advanced past the Version check.
	event := newMockEventWithCUDs(t, qNameCmdInitiateLeaveWorkspace, []istructs.ICUDRow{
		newInviteCUD(1, 42),
	})

	err := projectorFn(event, st, nil)
	require.ErrorIs(err, expectedErr)
	st.AssertCalled(t, "KeyBuilder", sys.Storage_Record, QNameCDocInvite)
}

func TestApplyInviteEvents_SequentialJoinEventsReplaceControllerInPLogOrder(t *testing.T) {
	const (
		subjectID       = istructs.RecordID(101)
		firstInviteID   = istructs.RecordID(201)
		secondInviteID  = istructs.RecordID(202)
		thirdInviteID   = istructs.RecordID(203)
		canonicalEmail  = "jsmith@example.com"
		aliasEmail      = "j.smith@example.com"
		replacementMail = "john.smith@example.com"
	)

	firstRequests := projectInviteReplacement(t, joinReplacementInput{
		previousInviteID:    firstInviteID,
		previousInviteEmail: canonicalEmail,
		currentInviteID:     secondInviteID,
		currentInviteEmail:  aliasEmail,
		subjectID:           subjectID,
	})
	requireAtomicInviteReplacement(t, firstRequests, firstInviteID, secondInviteID, subjectID, aliasEmail)

	secondRequests := projectInviteReplacement(t, joinReplacementInput{
		previousInviteID:    secondInviteID,
		previousInviteEmail: aliasEmail,
		currentInviteID:     thirdInviteID,
		currentInviteEmail:  replacementMail,
		subjectID:           subjectID,
	})
	requireAtomicInviteReplacement(t, secondRequests, secondInviteID, thirdInviteID, subjectID, replacementMail)
}

func TestApplyInviteEvents_ReplayedJoinedInviteIsNoOp(t *testing.T) {
	const inviteID = istructs.RecordID(42)

	inviteKey := newApplyInviteEventsKeyBuilder()
	invite := &coreutils.MockStateValue{}
	invite.On("AsInt32", Field_State).Return(int32(State_Joined))

	st := &coreutils.MockState{}
	st.On("KeyBuilder", sys.Storage_Record, QNameCDocInvite).Return(inviteKey, nil).Once()
	st.On("MustExist", inviteKey).Return(invite, nil).Once()

	event := newMockEventWithCUDs(t, qNameCmdInitiateJoinWorkspace, []istructs.ICUDRow{
		newInviteCUD(1, inviteID),
	})
	event.On("ArgumentObject").Return(&coreutils.TestObject{
		Data: map[string]any{field_InviteID: inviteID},
	})

	require.NoError(t, applyInviteEvents(nil, nil, nil, smtp.Cfg{})(event, st, nil))
	st.AssertNumberOfCalls(t, "KeyBuilder", 1)
	st.AssertNumberOfCalls(t, "MustExist", 1)
}

func TestApplyInviteEvents_UnresolvedControllingInviteIsLoggedAndSkipped(t *testing.T) {
	const (
		workspaceID    = istructs.WSID(500)
		subjectID      = istructs.RecordID(101)
		inviteID       = istructs.RecordID(202)
		canonicalLogin = "jsmith@example.com"
		aliasEmail     = "j.smith@example.com"
	)

	currentInviteKey := newApplyInviteEventsKeyBuilder()
	subjectIndexKey := newApplyInviteEventsKeyBuilder()
	subjectKey := newApplyInviteEventsKeyBuilder()
	inviteIndexKey := newApplyInviteEventsKeyBuilder()

	currentInvite := &coreutils.MockStateValue{}
	currentInvite.On("AsInt32", Field_State).Return(int32(State_Invited)).Maybe()
	currentInvite.On("AsString", field_ActualLogin).Return(canonicalLogin).Maybe()
	currentInvite.On("AsString", Field_Email).Return(aliasEmail).Maybe()

	subjectIndex := &coreutils.MockStateValue{}
	subjectIndex.On("AsRecordID", Field_SubjectID).Return(subjectID).Maybe()

	subject := &coreutils.MockStateValue{}
	subject.On("AsBool", appdef.SystemField_IsActive).Return(true).Maybe()
	subject.On("AsString", Field_InviteEmail).Return("").Maybe()

	st := &coreutils.MockState{}
	st.On("KeyBuilder", sys.Storage_Record, QNameCDocInvite).Return(currentInviteKey, nil).Once()
	st.On("MustExist", currentInviteKey).Return(currentInvite, nil).Once()
	st.On("KeyBuilder", sys.Storage_View, QNameViewSubjectsIdx).Return(subjectIndexKey, nil).Once()
	st.On("CanExist", subjectIndexKey).Return(subjectIndex, true, nil).Once()
	st.On("KeyBuilder", sys.Storage_Record, QNameCDocSubject).Return(subjectKey, nil).Once()
	st.On("CanExist", subjectKey).Return(subject, true, nil).Once()
	st.On("KeyBuilder", sys.Storage_View, qNameViewInviteIndex).Return(inviteIndexKey, nil).Twice()
	st.On("CanExist", inviteIndexKey).Return(nil, false, nil).Twice()

	event := newMockEventWithCUDs(t, qNameCmdInitiateJoinWorkspace, []istructs.ICUDRow{
		newInviteCUD(1, inviteID),
	})
	event.On("ArgumentObject").Return(&coreutils.TestObject{
		Data: map[string]any{field_InviteID: inviteID},
	})
	event.On("Workspace").Return(workspaceID)

	logCapture := logger.StartCapture(t, logger.LogLevelError)
	require.NoError(t, applyInviteEvents(nil, nil, nil, smtp.Cfg{})(event, st, &coreutils.MockIntents{}))
	logCapture.HasLine(
		"skipping join event because the controlling invitation was not resolved",
		"workspace=500",
		"invite=202",
		`canonicalLogin="jsmith@example.com"`,
	)
	st.AssertExpectations(t)
	st.AssertNotCalled(t, "KeyBuilder", sys.Storage_Record, appdef.QNameCDocWorkspaceDescriptor)
}

type joinReplacementInput struct {
	previousInviteID    istructs.RecordID
	previousInviteEmail string
	currentInviteID     istructs.RecordID
	currentInviteEmail  string
	subjectID           istructs.RecordID
}

type capturedFederationRequest struct {
	path string
	body string
}

type capturedFederationRequests struct {
	sync.Mutex
	items []capturedFederationRequest
}

func (c *capturedFederationRequests) append(request capturedFederationRequest) {
	c.Lock()
	defer c.Unlock()
	c.items = append(c.items, request)
}

func (c *capturedFederationRequests) snapshot() []capturedFederationRequest {
	c.Lock()
	defer c.Unlock()
	return append([]capturedFederationRequest(nil), c.items...)
}

func projectInviteReplacement(t *testing.T, input joinReplacementInput) []capturedFederationRequest {
	t.Helper()
	const (
		workspaceID      = istructs.WSID(500)
		inviteeProfileID = istructs.WSID(600)
		workspaceName    = "Acme"
	)

	requests := &capturedFederationRequests{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		requests.append(capturedFederationRequest{path: r.URL.Path, body: string(body)})
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	fed, cleanup := federation.New(context.Background(), func() *url.URL {
		return serverURL
	}, func() int {
		return 0
	}, nil)
	t.Cleanup(cleanup)

	currentInviteKey := newApplyInviteEventsKeyBuilder()
	previousInviteKey := newApplyInviteEventsKeyBuilder()
	subjectIndexKey := newApplyInviteEventsKeyBuilder()
	subjectKey := newApplyInviteEventsKeyBuilder()
	inviteIndexKey := newApplyInviteEventsKeyBuilder()
	workspaceKey := newApplyInviteEventsKeyBuilder()

	currentInvite := &coreutils.MockStateValue{}
	currentInvite.On("AsInt32", Field_State).Return(int32(State_Invited)).Maybe()
	currentInvite.On("AsString", field_ActualLogin).Return("jsmith@example.com").Maybe()
	currentInvite.On("AsString", Field_Email).Return(input.currentInviteEmail).Maybe()
	currentInvite.On("AsString", Field_Roles).Return(applyInviteEventsTestReplacementRoles).Maybe()
	currentInvite.On("AsInt32", authnz.Field_SubjectKind).Return(int32(istructs.SubjectKind_User)).Maybe()
	currentInvite.On("AsInt64", Field_InviteeProfileWSID).Return(int64(inviteeProfileID)).Maybe()

	previousInvite := &coreutils.MockStateValue{}
	previousInvite.On("AsInt32", Field_State).Return(int32(State_Joined)).Maybe()
	previousInvite.On("AsString", Field_Email).Return(input.previousInviteEmail).Maybe()
	previousInvite.On("AsRecordID", field_SubjectID).Return(input.subjectID).Maybe()

	subjectIndex := &coreutils.MockStateValue{}
	subjectIndex.On("AsRecordID", Field_SubjectID).Return(input.subjectID).Maybe()

	subject := &coreutils.MockStateValue{}
	subject.On("AsBool", appdef.SystemField_IsActive).Return(true).Maybe()
	subject.On("AsString", applyInviteEventsTestInviteEmail).Return(input.previousInviteEmail).Maybe()

	inviteIndex := &coreutils.MockStateValue{}
	inviteIndex.On("AsRecordID", field_InviteID).Return(input.previousInviteID).Maybe()

	workspace := &coreutils.MockStateValue{}
	workspace.On("AsString", authnz.Field_WSName).Return(workspaceName).Maybe()

	st := &coreutils.MockState{}
	st.On("App").Return(istructs.AppQName_test1_app1).Maybe()
	st.On("KeyBuilder", sys.Storage_Record, QNameCDocInvite).Return(currentInviteKey, nil).Once()
	st.On("MustExist", currentInviteKey).Return(currentInvite, nil).Maybe()
	st.On("CanExist", currentInviteKey).Return(currentInvite, true, nil).Maybe()
	st.On("KeyBuilder", sys.Storage_View, QNameViewSubjectsIdx).Return(subjectIndexKey, nil).Maybe()
	st.On("CanExist", subjectIndexKey).Return(subjectIndex, true, nil).Maybe()
	st.On("MustExist", subjectIndexKey).Return(subjectIndex, nil).Maybe()
	st.On("KeyBuilder", sys.Storage_Record, QNameCDocSubject).Return(subjectKey, nil).Maybe()
	st.On("CanExist", subjectKey).Return(subject, true, nil).Maybe()
	st.On("MustExist", subjectKey).Return(subject, nil).Maybe()
	st.On("KeyBuilder", sys.Storage_View, qNameViewInviteIndex).Return(inviteIndexKey, nil).Maybe()
	st.On("CanExist", inviteIndexKey).Return(inviteIndex, true, nil).Maybe()
	st.On("MustExist", inviteIndexKey).Return(inviteIndex, nil).Maybe()
	st.On("KeyBuilder", sys.Storage_Record, QNameCDocInvite).Return(previousInviteKey, nil).Maybe()
	st.On("MustExist", previousInviteKey).Return(previousInvite, nil).Maybe()
	st.On("CanExist", previousInviteKey).Return(previousInvite, true, nil).Maybe()
	st.On("KeyBuilder", sys.Storage_Record, appdef.QNameCDocWorkspaceDescriptor).Return(workspaceKey, nil).Maybe()
	st.On("MustExist", workspaceKey).Return(workspace, nil).Maybe()

	event := newMockEventWithCUDs(t, qNameCmdInitiateJoinWorkspace, []istructs.ICUDRow{
		newInviteCUD(1, input.currentInviteID),
	})
	event.On("ArgumentObject").Return(&coreutils.TestObject{
		Data: map[string]any{field_InviteID: input.currentInviteID},
	})
	event.On("Workspace").Return(workspaceID)

	tm := timeu.NewITime()
	tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, tm)
	require.NoError(t, applyInviteEvents(tm, fed, tokens, smtp.Cfg{})(event, st, nil))
	return requests.snapshot()
}

func newApplyInviteEventsKeyBuilder() *coreutils.MockStateKeyBuilder {
	key := &coreutils.MockStateKeyBuilder{}
	key.On("PutInt32", mock.Anything, mock.Anything).Return().Maybe()
	key.On("PutInt64", mock.Anything, mock.Anything).Return().Maybe()
	key.On("PutString", mock.Anything, mock.Anything).Return().Maybe()
	key.On("PutQName", mock.Anything, mock.Anything).Return().Maybe()
	key.On("PutRecordID", mock.Anything, mock.Anything).Return().Maybe()
	return key
}

func requireAtomicInviteReplacement(t *testing.T, requests []capturedFederationRequest, previousInviteID, currentInviteID, subjectID istructs.RecordID, currentInviteEmail string) {
	t.Helper()

	workspaceCUDRequests := make([]capturedFederationRequest, 0, 1)
	for _, request := range requests {
		require.NotContains(t, request.path, "DeactivateJoinedWorkspace")
		if strings.HasSuffix(request.path, "/c.sys.CUD") {
			workspaceCUDRequests = append(workspaceCUDRequests, request)
		}
	}
	require.Len(t, workspaceCUDRequests, 1, "replacement must use one workspace-local CUD")

	var body struct {
		CUDs []struct {
			ID     istructs.RecordID `json:"sys.ID"`
			Fields map[string]any    `json:"fields"`
		} `json:"cuds"`
	}
	require.NoError(t, json.Unmarshal([]byte(workspaceCUDRequests[0].body), &body))
	require.Len(t, body.CUDs, 3)

	cudsByID := make(map[istructs.RecordID]map[string]any, len(body.CUDs))
	for _, cud := range body.CUDs {
		cudsByID[cud.ID] = cud.Fields
		require.NotContains(t, cud.Fields, appdef.SystemField_IsActive)
	}

	previousFields := cudsByID[previousInviteID]
	require.Equal(t, float64(State_Cancelled), previousFields[Field_State])
	if writtenSubjectID, ok := previousFields[field_SubjectID]; ok {
		require.Equal(t, float64(subjectID), writtenSubjectID, "retired invitation must preserve its historical SubjectID")
	}

	currentFields := cudsByID[currentInviteID]
	require.Equal(t, float64(State_Joined), currentFields[Field_State])
	require.Equal(t, float64(subjectID), currentFields[field_SubjectID])

	subjectFields := cudsByID[subjectID]
	require.Equal(t, applyInviteEventsTestReplacementRoles, subjectFields[Field_Roles])
	require.Equal(t, currentInviteEmail, subjectFields[applyInviteEventsTestInviteEmail])
}
