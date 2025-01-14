/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state/smtptest"
	"github.com/voedger/voedger/pkg/sys/invite"
	it "github.com/voedger/voedger/pkg/vit"
)

var (
	initialRoles        = "initial.Roles"
	newRoles            = "new.Roles"
	inviteEmailTemplate = "text:" + strings.Join([]string{
		invite.EmailTemplatePlaceholder_VerificationCode,
		invite.EmailTemplatePlaceholder_InviteID,
		invite.EmailTemplatePlaceholder_WSID,
		invite.EmailTemplatePlaceholder_WSName,
		invite.EmailTemplatePlaceholder_Email,
	}, ";")
	inviteEmailSubject = "you are invited"
)

// impossible to use the test workspace again with the same login due of invite error `subject already exists`
func TestInvite_BasicUsage(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	wsName := "TestInvite_BasicUsage_ws"
	wsParams := it.SimpleWSParams(wsName)
	updateRolesEmailTemplate := "text:" + invite.EmailTemplatePlaceholder_Roles
	updateRolesEmailSubject := "your roles are updated"
	expireDatetime := vit.Now().UnixMilli()
	updatedRoles := "app1pkg.Updated"

	email1 := fmt.Sprintf("testinvite_basicusage_%d@123.com", vit.NextNumber())
	email2 := fmt.Sprintf("testinvite_basicusage_%d@123.com", vit.NextNumber())
	email3 := fmt.Sprintf("testinvite_basicusage_%d@123.com", vit.NextNumber())
	login1 := vit.SignUp(email1, "1", istructs.AppQName_test1_app1)
	login2 := vit.SignUp(email2, "1", istructs.AppQName_test1_app1)
	login1Prn := vit.SignIn(login1)
	login2Prn := vit.SignIn(login2)
	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
	ws := vit.CreateWorkspace(wsParams, prn)

	initiateUpdateInviteRoles := func(inviteID istructs.RecordID) {
		vit.PostWS(ws, "c.sys.InitiateUpdateInviteRoles", fmt.Sprintf(`{"args":{"InviteID":%d,"Roles":"%s","EmailTemplate":"%s","EmailSubject":"%s"}}`, inviteID, updatedRoles, updateRolesEmailTemplate, updateRolesEmailSubject))
	}

	findCDocInviteByID := func(inviteID istructs.RecordID) []interface{} {
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
	inviteID := InitiateInvitationByEMail(vit, ws, expireDatetime, email1, initialRoles, inviteEmailTemplate, inviteEmailSubject)
	inviteID2 := InitiateInvitationByEMail(vit, ws, expireDatetime, email2, initialRoles, inviteEmailTemplate, inviteEmailSubject)
	inviteID3 := InitiateInvitationByEMail(vit, ws, expireDatetime, email3, initialRoles, inviteEmailTemplate, inviteEmailSubject)

	// need to gather email first because
	actualEmails := []smtptest.Message{vit.CaptureEmail(), vit.CaptureEmail(), vit.CaptureEmail()}

	WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)
	WaitForInviteState(vit, ws, inviteID2, invite.State_ToBeInvited, invite.State_Invited)
	WaitForInviteState(vit, ws, inviteID3, invite.State_ToBeInvited, invite.State_Invited)

	cDocInvite := findCDocInviteByID(inviteID)

	require.Equal(email1, cDocInvite[1])
	require.Equal(email1, cDocInvite[2])
	require.Equal(initialRoles, cDocInvite[3])
	require.Equal(float64(expireDatetime), cDocInvite[4])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[7])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	//Check that emails were sent
	var verificationCodeEmail, verificationCodeEmail2, verificationCodeEmail3 string
	for _, actualEmail := range actualEmails {
		switch actualEmail.To[0] {
		case email1:
			verificationCodeEmail = actualEmail.Body[:6]
		case email2:
			verificationCodeEmail2 = actualEmail.Body[:6]
		case email3:
			verificationCodeEmail3 = actualEmail.Body[:6]
		}
	}
	expectedEmails := []smtptest.Message{
		{
			Subject: inviteEmailSubject,
			From:    it.TestSMTPCfg.GetFrom(),
			To:      []string{email1},
			Body:    fmt.Sprintf("%s;%d;%d;%s;%s", verificationCodeEmail, inviteID, ws.WSID, wsName, email1),
			CC:      []string{},
			BCC:     []string{},
		},
		{
			Subject: inviteEmailSubject,
			From:    it.TestSMTPCfg.GetFrom(),
			To:      []string{email2},
			Body:    fmt.Sprintf("%s;%d;%d;%s;%s", verificationCodeEmail2, inviteID2, ws.WSID, wsName, email2),
			CC:      []string{},
			BCC:     []string{},
		},
		{
			Subject: inviteEmailSubject,
			From:    it.TestSMTPCfg.GetFrom(),
			To:      []string{email3},
			Body:    fmt.Sprintf("%s;%d;%d;%s;%s", verificationCodeEmail3, inviteID3, ws.WSID, wsName, email3),
			CC:      []string{},
			BCC:     []string{},
		},
	}
	require.EqualValues(expectedEmails, actualEmails)

	cDocInvite = findCDocInviteByID(inviteID2)

	require.Equal(verificationCodeEmail2, cDocInvite[5])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	// overwrite roles is possible when the invite is not accepted yet
	verificationCodeEmail = testOverwriteRoles(t, vit, ws, email1, inviteID)

	//Cancel then invite it again (inviteID3)
	vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID3))
	WaitForInviteState(vit, ws, inviteID3, invite.State_ToBeCancelled, invite.State_Cancelled)
	InitiateInvitationByEMail(vit, ws, expireDatetime, email3, initialRoles, inviteEmailTemplate, inviteEmailSubject)
	_ = vit.CaptureEmail()
	WaitForInviteState(vit, ws, inviteID3, invite.State_ToBeInvited, invite.State_Invited)

	//Join workspaces
	InitiateJoinWorkspace(vit, ws, inviteID, login1Prn, verificationCodeEmail)
	InitiateJoinWorkspace(vit, ws, inviteID2, login2Prn, verificationCodeEmail2)

	// State_ToBeJoined will be set for a very short period of time so let's do not catch it
	WaitForInviteState(vit, ws, inviteID, invite.State_ToBeJoined, invite.State_Joined)
	WaitForInviteState(vit, ws, inviteID2, invite.State_ToBeJoined, invite.State_Joined)

	cDocInvite = findCDocInviteByID(inviteID2)

	require.Equal(float64(login2Prn.ProfileWSID), cDocInvite[10])
	require.Equal(float64(istructs.SubjectKind_User), cDocInvite[0])
	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	cDocJoinedWorkspace := FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(vit, ws.WSID, login2Prn)

	require.Equal(initialRoles, cDocJoinedWorkspace.roles)
	require.Equal(wsName, cDocJoinedWorkspace.wsName)

	cDocSubject := findCDocSubjectByLogin(email1)

	require.Equal(email1, cDocSubject[0])
	require.Equal(float64(istructs.SubjectKind_User), cDocSubject[1])
	require.Equal(newRoles, cDocSubject[2]) // overwritten

	t.Run("reinivite the joined already -> error", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Email":"%s","Roles":"%s","ExpireDatetime":%d,"EmailTemplate":"%s","EmailSubject":"%s"}}`,
			email1, initialRoles, vit.Now().UnixMilli(), inviteEmailTemplate, inviteEmailSubject)
		vit.PostWS(ws, "c.sys.InitiateInvitationByEMail", body, coreutils.Expect400(invite.ErrSubjectAlreadyExists.Error()))
	})

	//Update roles
	initiateUpdateInviteRoles(inviteID)
	initiateUpdateInviteRoles(inviteID2)

	//Check that emails were sent
	require.Equal(updatedRoles, vit.CaptureEmail().Body)
	message := vit.CaptureEmail()
	require.Equal(updateRolesEmailSubject, message.Subject)
	require.Equal(it.TestSMTPCfg.GetFrom(), message.From)
	require.Equal([]string{email2}, message.To)
	require.Equal(updatedRoles, message.Body)

	WaitForInviteState(vit, ws, inviteID, invite.State_Joined, invite.State_ToUpdateRoles, invite.State_Joined)
	WaitForInviteState(vit, ws, inviteID2, invite.State_Joined, invite.State_ToUpdateRoles, invite.State_Joined)
	cDocInvite = findCDocInviteByID(inviteID)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	cDocSubject = findCDocSubjectByLogin(email2)

	require.Equal(updatedRoles, cDocSubject[2])

	//TODO Denis how to get WS by login? I want to check sys.JoinedWorkspace

	WaitForInviteState(vit, ws, inviteID, invite.State_ToBeJoined, invite.State_Joined)
	WaitForInviteState(vit, ws, inviteID2, invite.State_ToBeJoined, invite.State_Joined)

	//Cancel accepted invite
	vit.PostWS(ws, "c.sys.InitiateCancelAcceptedInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))

	// State_ToBeCancelled will be set for a veri short period of time so let's do not catch it
	WaitForInviteState(vit, ws, inviteID, invite.State_ToBeCancelled, invite.State_Cancelled)

	cDocInvite = findCDocInviteByID(inviteID)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	cDocSubject = findCDocSubjectByLogin(email1)

	require.False(cDocSubject[4].(bool))

	cDocInvite = findCDocInviteByID(inviteID)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	//Leave workspace
	vit.PostWS(ws, "c.sys.InitiateLeaveWorkspace", "{}", coreutils.WithAuthorizeBy(login2Prn.Token))

	WaitForInviteState(vit, ws, inviteID2, invite.State_ToBeLeft, invite.State_Left)

	cDocInvite = findCDocInviteByID(inviteID2)

	require.Equal(float64(vit.Now().UnixMilli()), cDocInvite[8])

	cDocSubject = findCDocSubjectByLogin(email2)

	require.False(cDocSubject[4].(bool))

	//TODO check InviteeProfile joined workspace

	//Re-invite
	newRoles := "new.roles"
	InitiateInvitationByEMail(vit, ws, expireDatetime, email2, newRoles, inviteEmailTemplate, inviteEmailSubject)
	log.Println(vit.CaptureEmail().Body)
	WaitForInviteState(vit, ws, inviteID2, invite.State_Left, invite.State_Invited)
	cDocInvite = findCDocInviteByID(inviteID2)
	require.Equal(newRoles, cDocInvite[3].(string))
}

func TestCancelSentInvite(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	email := fmt.Sprintf("testcancelsentinvite_%d@123.com", vit.NextNumber())
	login := vit.SignUp(email, "1", istructs.AppQName_test1_app1)
	loginPrn := vit.SignIn(login)
	wsParams := it.SimpleWSParams("TestCancelSentInvite_ws")
	ws := vit.CreateWorkspace(wsParams, loginPrn)

	t.Run("basic usage", func(t *testing.T) {
		inviteID := InitiateInvitationByEMail(vit, ws, vit.Now().UnixMilli(), email, initialRoles, inviteEmailTemplate, inviteEmailSubject)
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)

		//Read it for successful vit tear down
		vit.CaptureEmail()

		vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, inviteID))
		WaitForInviteState(vit, ws, inviteID, invite.State_ToBeCancelled, invite.State_Cancelled)
	})
	t.Run("invite not exists -> 400 bad request", func(t *testing.T) {
		vit.PostWS(ws, "c.sys.CancelSentInvite", fmt.Sprintf(`{"args":{"InviteID":%d}}`, istructs.NonExistingRecordID), coreutils.Expect400RefIntegrity_Existence())
	})
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
	vit.PostWS(parentWS, "c.sys.CUD", cudBody, coreutils.Expect403(), coreutils.WithAuthorizeBy(newPrn.Token))

	// make this new foreign login a subject in the existing workspace
	body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "sys.Subject","Login": "%s","SubjectKind":%d,"Roles": "%s","ProfileWSID":%d}}]}`,
		newLoginName, istructs.SubjectKind_User, iauthnz.QNameRoleWorkspaceOwner, newPrn.ProfileWSID)
	cdocSubjectID := vit.PostWS(parentWS, "c.sys.CUD", body).NewID()

	// now the foreign login could work in the workspace
	vit.PostWS(parentWS, "c.sys.CUD", cudBody, coreutils.WithAuthorizeBy(newPrn.Token))

	// deactivate cdoc.Subject
	body = fmt.Sprintf(`{"cuds": [{"sys.ID": %d,"fields": {"sys.IsActive": false}}]}`, cdocSubjectID)
	vit.PostWS(parentWS, "c.sys.CUD", body)

	// try again to work in the foreign workspace -> should fail
	vit.PostWS(parentWS, "c.sys.CUD", cudBody, coreutils.WithAuthorizeBy(newPrn.Token), coreutils.Expect403())
}

func testOverwriteRoles(t *testing.T, vit *it.VIT, ws *it.AppWorkspace, email string, inviteID istructs.RecordID) (verificationCode string) {
	require := require.New(t)

	// reinvite when invitation is not accepted yet -> roles must be overwritten
	newInviteID := InitiateInvitationByEMail(vit, ws, vit.Now().UnixMilli(), email, newRoles, inviteEmailTemplate, inviteEmailSubject)
	require.Zero(newInviteID)
	WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)
	actualEmail := vit.CaptureEmail()
	verificationCode = actualEmail.Body[:6]

	// expect roles are overwritten in cdoc.sys.Invite
	body := fmt.Sprintf(`{"args":{"Schema":"sys.Invite","ID":%d},"elements":[{"fields":["Roles"]}]}`, inviteID)
	resp := vit.PostWS(ws, "q.sys.Collection", body)
	require.Equal(newRoles, resp.SectionRow()[0].(string))

	return verificationCode
}

func TestRejectInvitationOnDifferentLogin(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	email := fmt.Sprintf("testcancelsentinvite_%d@123.com", vit.NextNumber())
	login := vit.SignUp(email, "1", istructs.AppQName_test1_app1)
	loginPrn := vit.SignIn(login)
	wsParams := it.SimpleWSParams("TestCancelSentInvite_ws")
	ws := vit.CreateWorkspace(wsParams, loginPrn)
	inviteID := InitiateInvitationByEMail(vit, ws, vit.Now().UnixMilli(), email, initialRoles, inviteEmailTemplate, inviteEmailSubject)
	actualEmail := vit.CaptureEmail()
	verificationCode := actualEmail.Body[:6]
	WaitForInviteState(vit, ws, inviteID, invite.State_ToBeInvited, invite.State_Invited)

	// simulate accepting invitation by different login
	differentLogin := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
	InitiateJoinWorkspace(vit, ws, inviteID, differentLogin, verificationCode,
		coreutils.Expect400(fmt.Sprintf("invitation was sent to %s but current login is login", email)))
}
