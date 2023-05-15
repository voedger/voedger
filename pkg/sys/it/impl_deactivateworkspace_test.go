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
	startTime := time.Now()
	for time.Since(startTime) < 10*time.Second {
		ws = vit.WaitForWorkspace(ws.Name, prn1)
		if !ws.IsActive {
			break
		}
	}
	require.False(ws.IsActive)

	// try to exec something in a deactivated workspace
	body := `{"cuds":[{"fields":{"sys.QName":"sys.computers","sys.ID":1}}]}`
	vit.PostWS(ws, "c.sys.CUD", body)
}

func Test_DeactivateWorkspace(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	wsName1 := vit.NextName()
	wsName2 := vit.NextName()
	prn1 := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
	prn2 := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail2)
	wsp := it.WSParams{
		Name:         wsName1,
		Kind:         it.QNameTestWSKind,
		InitDataJSON: `{"IntFld":42}`,
		ClusterID:    istructs.MainClusterID,
	}

	ws1 := vit.CreateWorkspace(wsp, prn1)

	wsp.Name = wsName2

	// join ws2 to ws1
	expireDatetime := vit.Now().UnixMilli()
	roleOwner := iauthnz.QNameRoleWorkspaceOwner.String()
	updateRolesEmailTemplate := "text:" + invite.EmailTemplatePlaceholder_Roles
	updateRolesEmailSubject := "your roles are updated"
	inviteID := InitiateInvitationByEMail(vit, ws1, expireDatetime, it.TestEmail2, roleOwner, updateRolesEmailTemplate, updateRolesEmailSubject)

	vit.ExpectEmail().Capture()
	// vit.ExpectEmail().Capture()

	WaitForInviteState(vit, ws1, invite.State_Invited, inviteID)

	expireDatetimeStr := strconv.FormatInt(expireDatetime, 10)
	verificationCode := expireDatetimeStr[len(expireDatetimeStr)-6:]
	InitiateJoinWorkspace(vit, ws1, inviteID, it.TestEmail2, verificationCode)

	WaitForInviteState(vit, ws1, invite.State_Joined, inviteID)

	// check prn2 could work in ws1
	body := `{"cuds":[{"fields":{"sys.QName":"sys.computers","sys.ID":1}}]}`
	vit.PostWS(ws1, "c.sys.CUD", body, coreutils.WithAuthorizeBy(prn2.Token))

	vit.PostWS(ws1, "c.sys.DeactivateWorkspace", "{}")

	for {
		ws := vit.WaitForWorkspace(ws1.Name, prn1)
		if !ws.IsActive {
			break
		}
	}
	time.Sleep(100 * time.Hour)
}
