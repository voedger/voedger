/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/invite"
	it "github.com/voedger/voedger/pkg/vit"
)

var (
	initialRoles        = "app1pkg.LimitedAccessRole"
	newRoles            = "app1pkg.SpecialAPITokenRole"
	inviteEmailTemplate = "text:" + strings.Join([]string{
		invite.EmailTemplatePlaceholder_VerificationCode,
		invite.EmailTemplatePlaceholder_InviteID,
		invite.EmailTemplatePlaceholder_WSID,
		invite.EmailTemplatePlaceholder_WSName,
		invite.EmailTemplatePlaceholder_Email,
	}, ";")
	inviteEmailSubject = "you are invited"
)

func TestCancelSentInvite_NonExisting(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	owner := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
	ws := vit.CreateWorkspace(it.SimpleWSParams("TestCancelSentInvite_NonExisting_ws"), owner)
	vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, istructs.NonExistingRecordID), it.Expect400RefIntegrity_Existence())
}

func TestInactiveCDocSubject(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// sign up a new login
	newLoginName := vit.NextName()
	newLogin := vit.SignUp(newLoginName, "1", istructs.AppQName_test1_app1)
	newPrn := vit.SignIn(newLogin)

	parentWS := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// try to execute an operation by the foreign login, expect 403
	cudBody := `{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "app1pkg.articles","name": "cola","article_manual": 1,"article_hash": 2,"hideonhold": 3,"time_active": 4,"control_active": 5}}]}`
	vit.PostWS(parentWS, "c.sys.CUD", cudBody, httpu.Expect403(), httpu.WithAuthorizeBy(newPrn.Token))

	// make this new foreign login a subject in the existing workspace
	body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "sys.Subject","Login": "%s","SubjectKind":%d,"Roles": "%s","ProfileWSID":%d}}]}`,
		newLoginName, istructs.SubjectKind_User, iauthnz.QNameRoleWorkspaceOwner, newPrn.ProfileWSID)
	cdocSubjectID := vit.PostWS(parentWS, "c.sys.CUD", body).NewID()

	// now the foreign login could work in the workspace
	vit.PostWS(parentWS, "c.sys.CUD", cudBody, httpu.WithAuthorizeBy(newPrn.Token))

	// deactivate cdoc.Subject
	body = fmt.Sprintf(`{"cuds": [{"sys.ID": %d,"fields": {"sys.IsActive": false}}]}`, cdocSubjectID)
	vit.PostWS(parentWS, "c.sys.CUD", body)

	// try again to work in the foreign workspace -> should fail
	vit.PostWS(parentWS, "c.sys.CUD", cudBody, httpu.WithAuthorizeBy(newPrn.Token), httpu.Expect403())
}

func TestRecoverFromStuckInviteStates(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
	ws := vit.CreateWorkspace(it.SimpleWSParams("TestRecoverStuckStates_ws"), prn)

	t.Run("re-invite from State_ToBeInvited", func(t *testing.T) {
		email := fmt.Sprintf("stuck_tobeinvited_reinvite_%d@test.com", vit.NextNumber())
		inviteID := InitiateInvitationByEMail(vit, ws, vit.Now().UnixMilli(), email, initialRoles, inviteEmailTemplate, inviteEmailSubject)
		vit.CaptureEmail()
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)

		// Simulate stuck state by forcing State_ToBeInvited via direct CUD
		setInviteState(vit, ws, inviteID, invite.State_ToBeInvited)

		// Re-invite should succeed from State_ToBeInvited
		InitiateInvitationByEMail(vit, ws, vit.Now().UnixMilli(), email, initialRoles, inviteEmailTemplate, inviteEmailSubject)
		vit.CaptureEmail()
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)
	})

	t.Run("cancel from State_ToBeInvited", func(t *testing.T) {
		email := fmt.Sprintf("stuck_tobeinvited_cancel_%d@test.com", vit.NextNumber())
		inviteID := InitiateInvitationByEMail(vit, ws, vit.Now().UnixMilli(), email, initialRoles, inviteEmailTemplate, inviteEmailSubject)
		vit.CaptureEmail()
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)

		// Simulate stuck state
		setInviteState(vit, ws, inviteID, invite.State_ToBeInvited)

		// Cancel should succeed from State_ToBeInvited
		vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Cancelled)
	})

	t.Run("re-invite from State_ToBeJoined", func(t *testing.T) {
		email := fmt.Sprintf("stuck_tobejoined_reinvite_%d@test.com", vit.NextNumber())
		inviteID := InitiateInvitationByEMail(vit, ws, vit.Now().UnixMilli(), email, initialRoles, inviteEmailTemplate, inviteEmailSubject)
		vit.CaptureEmail()
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)

		// Simulate stuck state by forcing State_ToBeJoined via direct CUD
		setInviteState(vit, ws, inviteID, invite.State_ToBeJoined)

		// Re-invite should succeed from State_ToBeJoined
		InitiateInvitationByEMail(vit, ws, vit.Now().UnixMilli(), email, initialRoles, inviteEmailTemplate, inviteEmailSubject)
		vit.CaptureEmail()
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)
	})

	t.Run("cancel from State_ToBeJoined", func(t *testing.T) {
		email := fmt.Sprintf("stuck_tobejoined_cancel_%d@test.com", vit.NextNumber())
		inviteID := InitiateInvitationByEMail(vit, ws, vit.Now().UnixMilli(), email, initialRoles, inviteEmailTemplate, inviteEmailSubject)
		vit.CaptureEmail()
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)

		// Simulate stuck state
		setInviteState(vit, ws, inviteID, invite.State_ToBeJoined)

		// Cancel should succeed from State_ToBeJoined
		vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeJoined, invite.State_Cancelled)
	})
}

// setInviteState directly sets invite state via CUD (for testing stuck state recovery)
func setInviteState(vit *it.VIT, ws *it.AppWorkspace, inviteID istructs.RecordID, state invite.State) {
	vit.T.Helper()
	body := fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"State":%d}}]}`, inviteID, state)
	vit.PostWS(ws, "c.sys.CUD", body)
}

// TestInvite_VersionMarker asserts that every current invite command writes Version=1
// on its cdoc.sys.Invite CUD. ap.sys.ApplyInviteEvents uses this discriminator to skip
// pre-refactor events whose CUD has Version==0 (or no cdoc.sys.Invite CUD at all).
func TestInvite_VersionMarker(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	expireDatetime := vit.Now().UnixMilli()
	ownerPrn := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	getInviteVersion := func(ws *it.AppWorkspace, inviteID istructs.RecordID) int32 {
		row := vit.PostWS(ws, "q.sys.Collection", fmt.Sprintf(`
			{"args":{"Schema":"sys.Invite"},
			"elements":[{"fields":["Version","sys.ID"]}],
			"filters":[{"expr":"eq","args":{"field":"sys.ID","value":%d}}]}`, inviteID)).SectionRow(0)
		return int32(row[0].(float64))
	}

	t.Run("create+cancelsent+reinvite+join+updateroles+leave", func(t *testing.T) {
		ws := vit.CreateWorkspace(it.SimpleWSParams("TestInvite_VersionMarker_A_ws"), ownerPrn)
		email := fmt.Sprintf("invite_version_a_%d@test.com", vit.NextNumber())
		login := vit.SignUp(email, "1", istructs.AppQName_test1_app1)
		loginPrn := vit.SignIn(login)

		// 1. InitiateInvitationByEMail (create branch)
		inviteID := InitiateInvitationByEMail(vit, ws, expireDatetime, email, initialRoles, inviteEmailTemplate, inviteEmailSubject)
		vit.CaptureEmail()
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)
		require.Equal(int32(1), getInviteVersion(ws, inviteID), "after InitiateInvitationByEMail (create)")

		// 2. CancelSentInvite
		vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
		WaitForInviteState(vit, ws, inviteID, invite.State_Invited, invite.State_Cancelled)
		require.Equal(int32(1), getInviteVersion(ws, inviteID), "after CancelSentInvite")

		// 3. InitiateInvitationByEMail (re-invite update branch)
		InitiateInvitationByEMail(vit, ws, expireDatetime, email, initialRoles, inviteEmailTemplate, inviteEmailSubject)
		verificationCode := vit.CaptureEmail().Body[:6]
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)
		require.Equal(int32(1), getInviteVersion(ws, inviteID), "after InitiateInvitationByEMail (re-invite)")

		// 4. InitiateJoinWorkspace
		InitiateJoinWorkspace(vit, ws, inviteID, loginPrn, verificationCode)
		WaitForInviteState(vit, ws, inviteID, invite.State_Invited, invite.State_Joined)
		require.Equal(int32(1), getInviteVersion(ws, inviteID), "after InitiateJoinWorkspace")

		// 5. InitiateUpdateInviteRoles (state stays Joined)
		vit.PostWS(ws, "c.sys.InitiateUpdateInviteRoles", fmt.Sprintf(
			`{"args":{"InviteID":%d,"Roles":"%s","EmailTemplate":"%s","EmailSubject":"%s"}}`,
			inviteID, newRoles, inviteEmailTemplate, inviteEmailSubject))
		vit.CaptureEmail()
		require.Equal(int32(1), getInviteVersion(ws, inviteID), "after InitiateUpdateInviteRoles")

		// 6. InitiateLeaveWorkspace (called by invitee)
		vit.PostWS(ws, "c.sys.InitiateLeaveWorkspace", "{}", httpu.WithAuthorizeBy(loginPrn.Token))
		WaitForInviteState(vit, ws, inviteID, invite.State_Joined, invite.State_Left)
		require.Equal(int32(1), getInviteVersion(ws, inviteID), "after InitiateLeaveWorkspace")
	})

	t.Run("InitiateCancelAcceptedInvite", func(t *testing.T) {
		ws := vit.CreateWorkspace(it.SimpleWSParams("TestInvite_VersionMarker_B_ws"), ownerPrn)
		email := fmt.Sprintf("invite_version_b_%d@test.com", vit.NextNumber())
		login := vit.SignUp(email, "1", istructs.AppQName_test1_app1)
		loginPrn := vit.SignIn(login)

		inviteID := InitiateInvitationByEMail(vit, ws, expireDatetime, email, initialRoles, inviteEmailTemplate, inviteEmailSubject)
		verificationCode := vit.CaptureEmail().Body[:6]
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)

		InitiateJoinWorkspace(vit, ws, inviteID, loginPrn, verificationCode)
		WaitForInviteState(vit, ws, inviteID, invite.State_Invited, invite.State_Joined)

		vit.PostWS(ws, "c.sys.InitiateCancelAcceptedInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
		WaitForInviteState(vit, ws, inviteID, invite.State_Joined, invite.State_Cancelled)
		require.Equal(int32(1), getInviteVersion(ws, inviteID), "after InitiateCancelAcceptedInvite")
	})
}
