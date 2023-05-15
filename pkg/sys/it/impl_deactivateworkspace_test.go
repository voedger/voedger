/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package sys_it

import (
	"strconv"
	"testing"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/invite"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_DeactivateWorkspace(t *testing.T) {
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
	ws2 := vit.CreateWorkspace(wsp, prn2)

	// join ws2 to ws1
	expireDatetime := vit.Now().UnixMilli()
	initialRoles := "initial.Roles"
	updateRolesEmailTemplate := "text:" + invite.EmailTemplatePlaceholder_Roles
	updateRolesEmailSubject := "your roles are updated"
	inviteID1 := InitiateInvitationByEMail(vit, ws1, expireDatetime, it.TestEmail, initialRoles, updateRolesEmailTemplate, updateRolesEmailSubject)
	inviteID2 := InitiateInvitationByEMail(vit, ws2, expireDatetime, it.TestEmail2, initialRoles, updateRolesEmailTemplate, updateRolesEmailSubject)

	vit.ExpectEmail().Capture()
	vit.ExpectEmail().Capture()

	WaitForInviteState(vit, ws1, invite.State_Invited, inviteID1)
	WaitForInviteState(vit, ws2, invite.State_Invited, inviteID2)

	expireDatetimeStr := strconv.FormatInt(expireDatetime, 10)
	verificationCode := expireDatetimeStr[len(expireDatetimeStr)-6:]
	InitiateJoinWorkspace(vit, ws1, inviteID1, it.TestEmail, verificationCode)
	InitiateJoinWorkspace(vit, ws2, inviteID2, it.TestEmail2, verificationCode)

	vit.PostWS(ws1, "c.sys.DeactivateWorkspace", "{}")
}
