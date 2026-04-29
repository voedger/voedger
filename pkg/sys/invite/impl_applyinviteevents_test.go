/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Maxim Geraskin
 */

package invite

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
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
