/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package sys_it

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/invite"
	sysshared "github.com/voedger/voedger/pkg/sys/shared"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_DeactivateWorkspace(t *testing.T) {
	require := require.New(t)
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
	require.True(ws.IsActive)

	// deactivate workspace
	vit.PostWS(ws, "c.sys.DeactivateWorkspace", "{}")
	for {
		resp := vit.PostWS(ws, "q.sys.Collection", `{"args":{"Schema":"sys.WorkspaceDescriptor"},"elements":[{"fields":["Status"]}]}`)
		if int32(resp.SectionRow()[0].(float64)) == int32(sysshared.WorkspaceStatus_Inactive) {
			break
		}
	}
	// startTime := time.Now()
	// for time.Since(startTime) < 10*time.Second {
	// 	ws = vit.WaitForWorkspace(ws.Name, prn1)
	// 	if !ws.IsActive {
	// 		break
	// 	}
	// }

	// try to exec something in a deactivated workspace
	body := `{"cuds":[{"fields":{"sys.QName":"sys.computers","sys.ID":1}}]}`
	vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect403())
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

	vit.ExpectEmail().Capture()

	WaitForInviteState(vit, newWS, invite.State_Invited, inviteID)

	expireDatetimeStr := strconv.FormatInt(expireDatetime, 10)
	verificationCode := expireDatetimeStr[len(expireDatetimeStr)-6:]
	InitiateJoinWorkspace(vit, newWS, inviteID, it.TestEmail2, verificationCode)

	WaitForInviteState(vit, newWS, invite.State_Joined, inviteID)

	// check prn2 could work in ws1
	body = `{"cuds":[{"fields":{"sys.QName":"sys.computers","sys.ID":1}}]}`
	vit.PostWS(newWS, "c.sys.CUD", body, coreutils.WithAuthorizeBy(prn2.Token))

	vit.PostWS(newWS, "c.sys.DeactivateWorkspace", "{}")

	for {
		ws := vit.WaitForWorkspace(newWS.Name, prn1)
		if !ws.IsActive {
			break
		}
	}
	time.Sleep(100 * time.Hour)

	// now check that cdoc.sys.JoinedWorkspace.IsActive == false
	cDocsJoinedWorkspace := FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(vit, newWS.WSID, it.TestEmail2)
	require.Len(cDocsJoinedWorkspace, 1)
}
