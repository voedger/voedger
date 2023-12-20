/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state/smtptest"
	"github.com/voedger/voedger/pkg/sys/invite"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestMain(t *testing.T) {
	gmtTimeLoc := time.FixedZone("GMT", 0)
	s := time.Now().In(gmtTimeLoc).Format("Mon, 02 Jan 2006 15:04:05.000 GMT")
	log.Println(s)
}

var (
	initialRoles        = "initial.Roles"
	inviteEmailTemplate = "text:" + strings.Join([]string{
		invite.EmailTemplatePlaceholder_VerificationCode,
		invite.EmailTemplatePlaceholder_InviteID,
		invite.EmailTemplatePlaceholder_WSID,
		invite.EmailTemplatePlaceholder_WSName,
		invite.EmailTemplatePlaceholder_Email,
	}, ";")
	inviteEmailSubject = "you are invited"
)

func TestInvite_BasicUsage(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	wsName := "test_ws"
	ws := vit.WS(istructs.AppQName_test1_app1, wsName)
	updateRolesEmailTemplate := "text:" + invite.EmailTemplatePlaceholder_Roles
	updateRolesEmailSubject := "your roles are updated"
	expireDatetime := vit.Now().UnixMilli()
	updatedRoles := "updated.Roles"

	initiateUpdateInviteRoles := func(inviteID int64) {
		vit.PostWS(ws, "c.sys.InitiateUpdateInviteRoles", fmt.Sprintf(`{"args":{"InviteID":%d,"Roles":"%s","EmailTemplate":"%s","EmailSubject":"%s"}}`, inviteID, updatedRoles, updateRolesEmailTemplate, updateRolesEmailSubject))
	}

	findCDocInviteByID := func(inviteID int64) []interface{} {
		return vit.PostWS(ws, "q.sys.Collection", fmt.Sprintf(`
			{"args":{"Schema":"sys.Invite"},
			"elements":[{"fields":[
				"SubjectKind",
				"Login",
				"Email",
				"Roles",
				"ExpireDatetime",
				"VerificationCode",
				"State",
				"Created",
				"Updated",
				"SubjectID",
				"InviteeProfileWSID",
				"ActualLogin",
				"sys.ID"
			]}],
			"filters":[{"expr":"eq","args":{"field":"sys.ID","value":%d}}]}`, inviteID)).SectionRow(0)
	}

	findCDocSubjectByLogin := func(login string) []interface{} {
		return vit.PostWS(ws, "q.sys.Collection", fmt.Sprintf(`
			{"args":{"Schema":"sys.Subject"},
			"elements":[{"fields":[
				"Login",
				"SubjectKind",
				"Roles",
				"sys.ID",
				"sys.IsActive"
			]}],
			"filters":[{"expr":"eq","args":{"field":"Login","value":"%s"}}]}`, login)).SectionRow(0)
	}

	//Invite existing users
	inviteID := InitiateInvitationByEMail(vit, ws, expireDatetime, it.TestEmail, initialRoles, inviteEmailTemplate, inviteEmailSubject)
	inviteID2 := InitiateInvitationByEMail(vit, ws, expireDatetime, it.TestEmail2, initialRoles, inviteEmailTemplate, inviteEmailSubject)
	inviteID3 := InitiateInvitationByEMail(vit, ws, expireDatetime, it.TestEmail3, initialRoles, inviteEmailTemplate, inviteEmailSubject)

	// need to gather email first because
	actualEmails := []smtptest.Message{vit.CaptureEmail(), vit.CaptureEmail(), vit.CaptureEmail()}

	// State ToBeInvite exists for a very small period of time so let's do not catch it
	WaitForInviteState(vit, ws, invite.State_Invited, inviteID)
	WaitForInviteState(vit, ws, invite.State_Invited, inviteID2)
	WaitForInviteState(vit, ws, invite.State_Invited, inviteID3)

	cDocInvite := findCDocInviteByID(inviteID)

	require.Equal(it.TestEmail, cDocInvite[1])
	require.Equal(it.TestEmail, cDocInvite[2])
	require.Equal(initialRoles, cDocInvite[3])
	require.Equal(float64(expireDatetime), cDocInvite[4])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[7])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	//Check that emails were sent
	var verificationCodeEmail, verificationCodeEmail2, verificationCodeEmail3 string
	for _, actualEmail := range actualEmails {
		switch actualEmail.To[0] {
		case it.TestEmail:
			verificationCodeEmail = actualEmail.Body[:6]
		case it.TestEmail2:
			verificationCodeEmail2 = actualEmail.Body[:6]
		case it.TestEmail3:
			verificationCodeEmail3 = actualEmail.Body[:6]
		}
	}
	expectedEmails := []smtptest.Message{
		{
			Subject: inviteEmailSubject,
			From:    it.TestSMTPCfg.GetFrom(),
			To:      []string{it.TestEmail},
			Body:    fmt.Sprintf("%s;%d;%d;%s;%s", verificationCodeEmail, inviteID, ws.WSID, wsName, it.TestEmail),
			CC:      []string{},
			BCC:     []string{},
		},
		{
			Subject: inviteEmailSubject,
			From:    it.TestSMTPCfg.GetFrom(),
			To:      []string{it.TestEmail2},
			Body:    fmt.Sprintf("%s;%d;%d;%s;%s", verificationCodeEmail2, inviteID2, ws.WSID, wsName, it.TestEmail2),
			CC:      []string{},
			BCC:     []string{},
		},
		{
			Subject: inviteEmailSubject,
			From:    it.TestSMTPCfg.GetFrom(),
			To:      []string{it.TestEmail3},
			Body:    fmt.Sprintf("%s;%d;%d;%s;%s", verificationCodeEmail3, inviteID3, ws.WSID, wsName, it.TestEmail3),
			CC:      []string{},
			BCC:     []string{},
		},
	}
	require.EqualValues(expectedEmails, actualEmails)

	cDocInvite = findCDocInviteByID(inviteID2)

	require.Equal(verificationCodeEmail2, cDocInvite[5])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	//Cancel then invite it again (inviteID3)
	vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID3))
	WaitForInviteState(vit, ws, invite.State_Cancelled, inviteID3)
	InitiateInvitationByEMail(vit, ws, expireDatetime, it.TestEmail3, initialRoles, inviteEmailTemplate, inviteEmailSubject)
	// WaitForInviteState(vit, ws, invite.State_ToBeInvited, inviteID3)
	_ = vit.CaptureEmail()
	WaitForInviteState(vit, ws, invite.State_Invited, inviteID3)

	//Join workspaces
	InitiateJoinWorkspace(vit, ws, inviteID, it.TestEmail, verificationCodeEmail)
	InitiateJoinWorkspace(vit, ws, inviteID2, it.TestEmail2, verificationCodeEmail2)

	WaitForInviteState(vit, ws, invite.State_ToBeJoined, inviteID)
	WaitForInviteState(vit, ws, invite.State_ToBeJoined, inviteID2)

	cDocInvite = findCDocInviteByID(inviteID2)

	require.Equal(float64(vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail2).ProfileWSID), cDocInvite[10])
	require.Equal(float64(istructs.SubjectKind_User), cDocInvite[0])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	WaitForInviteState(vit, ws, invite.State_Joined, inviteID)
	WaitForInviteState(vit, ws, invite.State_Joined, inviteID2)

	cDocJoinedWorkspace := FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(vit, ws.WSID, it.TestEmail2)

	require.Equal(initialRoles, cDocJoinedWorkspace.roles)
	require.Equal(wsName, cDocJoinedWorkspace.wsName)

	cDocSubject := findCDocSubjectByLogin(it.TestEmail)

	require.Equal(it.TestEmail, cDocSubject[0])
	require.Equal(float64(istructs.SubjectKind_User), cDocSubject[1])
	require.Equal(initialRoles, cDocSubject[2])

	//Update roles
	initiateUpdateInviteRoles(inviteID)
	initiateUpdateInviteRoles(inviteID2)

	WaitForInviteState(vit, ws, invite.State_ToUpdateRoles, inviteID)
	WaitForInviteState(vit, ws, invite.State_ToUpdateRoles, inviteID2)
	cDocInvite = findCDocInviteByID(inviteID)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	//Check that emails were send
	require.Equal(updatedRoles, vit.CaptureEmail().Body)
	message := vit.CaptureEmail()
	require.Equal(updateRolesEmailSubject, message.Subject)
	require.Equal(it.TestSMTPCfg.GetFrom(), message.From)
	require.Equal([]string{it.TestEmail2}, message.To)
	require.Equal(updatedRoles, message.Body)

	cDocSubject = findCDocSubjectByLogin(it.TestEmail2)

	require.Equal(updatedRoles, cDocSubject[2])

	//TODO Denis how to get WS by login? I want to check sys.JoinedWorkspace

	WaitForInviteState(vit, ws, invite.State_Joined, inviteID)
	WaitForInviteState(vit, ws, invite.State_Joined, inviteID2)

	//Cancel accepted invite
	vit.PostWS(ws, "c.sys.InitiateCancelAcceptedInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))

	WaitForInviteState(vit, ws, invite.State_ToBeCancelled, inviteID)

	cDocInvite = findCDocInviteByID(inviteID)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	WaitForInviteState(vit, ws, invite.State_Cancelled, inviteID)

	cDocSubject = findCDocSubjectByLogin(it.TestEmail)

	require.False(cDocSubject[4].(bool))

	cDocInvite = findCDocInviteByID(inviteID)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	//Leave workspace
	vit.PostWS(ws, "c.sys.InitiateLeaveWorkspace", "{}", coreutils.WithAuthorizeBy(vit.GetPrincipal(ws.Owner.AppQName, it.TestEmail2).Token))

	WaitForInviteState(vit, ws, invite.State_ToBeLeft, inviteID2)

	cDocInvite = findCDocInviteByID(inviteID2)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	WaitForInviteState(vit, ws, invite.State_Left, inviteID2)

	cDocSubject = findCDocSubjectByLogin(it.TestEmail2)

	require.False(cDocSubject[4].(bool))

	//TODO check InviteeProfile joined workspace

}

func TestCancelSentInvite(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("basic usage", func(t *testing.T) {
		inviteID := InitiateInvitationByEMail(vit, ws, 1674751138000, "user@acme.com", initialRoles, inviteEmailTemplate, inviteEmailSubject)

		//Read it for successful vit tear down
		vit.CaptureEmail()

		vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
		WaitForInviteState(vit, ws, invite.State_Cancelled, inviteID)
	})
	t.Run("invite not exists -> 400 bad request", func(t *testing.T) {
		vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, -100), coreutils.Expect400RefIntegrity_Existence())
	})
}
