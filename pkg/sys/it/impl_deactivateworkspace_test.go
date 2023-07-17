/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/invite"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_InitiateDeactivateWorkspace(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	wsName := vit.NextName()

	prn1 := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
	wsp := it.WSParams{
		Name:         wsName,
		Kind:         it.QNameTestWSKind,
		InitDataJSON: `{"IntFld":42}`,
		ClusterID:    istructs.MainClusterID,
	}

	ws := vit.CreateWorkspace(wsp, prn1)

	// initiate deactivate workspace
	vit.PostWS(ws, "c.sys.InitiateDeactivateWorkspace", "{}")
	waitForDeactivate(vit, ws)

	// 410 Gone on work in an inactive workspace
	bodyCmd := `{"cuds":[{"fields":{"sys.QName":"sys.computers","sys.ID":1}}]}`
	vit.PostWS(ws, "c.sys.CUD", bodyCmd, coreutils.Expect410()).Println()
	bodyQry := `{"args":{"Schema":"sys.WorkspaceDescriptor"},"elements":[{"fields":["Status"]}]}`
	vit.PostWS(ws, "q.sys.Collection", bodyQry, coreutils.Expect410()).Println()

	// still able to work in an inactive workspace with the system token
	sysToken := vit.GetSystemPrincipal(istructs.AppQName_test1_app1)
	vit.PostWS(ws, "q.sys.Collection", bodyQry, coreutils.WithAuthorizeBy(sysToken.Token))
	vit.PostWS(ws, "c.sys.CUD", bodyCmd, coreutils.WithAuthorizeBy(sysToken.Token))

	// 409 conflict on deactivate an already deactivated worksace
	vit.PostWS(ws, "c.sys.InitiateDeactivateWorkspace", "{}", coreutils.WithAuthorizeBy(sysToken.Token), coreutils.Expect409())
}

func waitForDeactivate(vit *it.VIT, ws *it.AppWorkspace) {
	deadline := time.Now().Add(time.Second)
	if coreutils.IsDebug() {
		deadline = deadline.Add(time.Hour)
	}
	for time.Now().Before(deadline) {
		resp := vit.PostWSSys(ws, "q.sys.Collection", `{"args":{"Schema":"sys.WorkspaceDescriptor"},"elements":[{"fields":["Status"]}]}`)
		if int32(resp.SectionRow()[0].(float64)) == int32(authnz.WorkspaceStatus_Inactive) {
			break
		}
		time.Sleep(awaitTime)
	}
}

func TestDeactivateJoinedWorkspace(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	wsName1 := vit.NextName()
	prn1 := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
	prn2 := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail2)
	wsp := it.WSParams{
		Name:         wsName1,
		Kind:         it.QNameTestWSKind,
		InitDataJSON: `{"IntFld":42}`,
		ClusterID:    istructs.MainClusterID,
	}

	newWS := vit.CreateWorkspace(wsp, prn1)

	// check prn2 could not work in ws1
	body := `{"cuds":[{"fields":{"sys.QName":"sys.computers","sys.ID":1}}]}`
	vit.PostWS(newWS, "c.sys.CUD", body, coreutils.WithAuthorizeBy(prn2.Token), coreutils.Expect403())

	// join login TestEmail2 to ws1
	expireDatetime := vit.Now().UnixMilli()
	roleOwner := iauthnz.QNameRoleWorkspaceOwner.String()
	updateRolesEmailTemplate := "text:" + invite.EmailTemplatePlaceholder_Roles
	updateRolesEmailSubject := "your roles are updated"
	inviteID := InitiateInvitationByEMail(vit, newWS, expireDatetime, it.TestEmail2, roleOwner, updateRolesEmailTemplate, updateRolesEmailSubject)
	vit.CaptureEmail()
	WaitForInviteState(vit, newWS, invite.State_Invited, inviteID)
	expireDatetimeStr := strconv.FormatInt(expireDatetime, 10)
	verificationCode := expireDatetimeStr[len(expireDatetimeStr)-6:]
	InitiateJoinWorkspace(vit, newWS, inviteID, it.TestEmail2, verificationCode)
	WaitForInviteState(vit, newWS, invite.State_Joined, inviteID)

	// check prn2 could work in ws1
	body = `{"cuds":[{"fields":{"sys.QName":"sys.computers","sys.ID":1}}]}`
	vit.PostWS(newWS, "c.sys.CUD", body, coreutils.WithAuthorizeBy(prn2.Token))

	// deactivate
	vit.PostWS(newWS, "c.sys.InitiateDeactivateWorkspace", "{}")
	waitForDeactivate(vit, newWS)

	// check cdoc.sys.JoinedWorkspace.IsActive == false
	joinedWorkspace := FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(vit, newWS.WSID, it.TestEmail2)
	require.False(joinedWorkspace.isActive)

	// check appWS/cdoc.sys.WorkspaceID.IsActive == false
	wsidOfCDocWorkspaceID := coreutils.GetPseudoWSID(prn1.ProfileWSID, newWS.Name, istructs.MainClusterID)
	body = fmt.Sprintf(`{"args":{"Query":"select IDOfCDocWorkspaceID from sys.WorkspaceIDIdx where OwnerWSID = %d and WSName = '%s'"}, "elements":[{"fields":["Result"]}]}`,
		prn1.ProfileWSID, newWS.Name)
	sysToken := vit.GetSystemPrincipal(istructs.AppQName_test1_app1)
	resp := vit.PostApp(istructs.AppQName_test1_app1, wsidOfCDocWorkspaceID, "q.sys.SqlQuery", body, coreutils.WithAuthorizeBy(sysToken.Token))
	viewWorkspaceIDIdx := map[string]interface{}{}
	require.NoError(json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &viewWorkspaceIDIdx))
	idOfCDocWorkspaceID := int64(viewWorkspaceIDIdx["IDOfCDocWorkspaceID"].(float64))
	body = fmt.Sprintf(`{"args":{"ID": %d},"elements":[{"fields": ["Result"]}]}`, int64(idOfCDocWorkspaceID))
	resp = vit.PostApp(istructs.AppQName_test1_app1, wsidOfCDocWorkspaceID, "q.sys.CDoc", body, coreutils.WithAuthorizeBy(sysToken.Token))
	jsonBytes := []byte(resp.SectionRow()[0].(string))
	cdocWorkspaceID := map[string]interface{}{}
	require.Nil(json.Unmarshal(jsonBytes, &cdocWorkspaceID))
	require.False(cdocWorkspaceID[appdef.SystemField_IsActive].(bool))
}
