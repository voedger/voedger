/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/vit"
)

func InitiateEmailVerification(vit *vit.VIT, prn *vit.Principal, entity appdef.QName, field, email string, targetWSID istructs.WSID, opts ...coreutils.ReqOptFunc) (token, code string) {
	return InitiateEmailVerificationFunc(vit, func() *coreutils.FuncResponse {
		body := fmt.Sprintf(`{"args":{"Entity":"%s","Field":"%s","Email":"%s","TargetWSID":%d},"elements":[{"fields":["VerificationToken"]}]}`, entity, field, email, targetWSID)
		return vit.PostApp(prn.AppQName, prn.ProfileWSID, "q.sys.InitiateEmailVerification", body, opts...)
	})
}

func InitiateEmailVerificationFunc(vit *vit.VIT, f func() *coreutils.FuncResponse) (token, code string) {
	resp := f()
	emailMessage := vit.CaptureEmail()
	r := regexp.MustCompile(`(?P<code>\d{6})`)
	matches := r.FindStringSubmatch(emailMessage.Body)
	code = matches[0]
	token = resp.SectionRow()[0].(string)
	return
}

func WaitForIndexOffset(vit *vit.VIT, ws *vit.AppWorkspace, index appdef.QName, offset int64) {
	type entity struct {
		Last int64 `json:"Last"`
	}

	body := fmt.Sprintf(`
			{
				"args":{"Query":"select LastOffset from %s where Year = %d and DayOfYear = %d"},
				"elements":[{"fields":["Result"]}]
			}`, index, vit.Now().Year(), vit.Now().YearDay())

	deadline := time.Now().Add(time.Second)

	for time.Now().Before(deadline) {
		resp := vit.PostWS(ws, "q.sys.SqlQuery", body)
		if resp.IsEmpty() {
			time.Sleep(awaitTime)
			continue
		}

		e := new(entity)

		err := json.Unmarshal([]byte(resp.SectionRow(0)[0].(string)), e)
		if err != nil {
			logger.Error(err)
		}
		if e.Last == offset {
			break
		}
	}
}

func InitiateInvitationByEMail(vit *vit.VIT, ws *vit.AppWorkspace, expireDatetime int64, email, initialRoles, inviteEmailTemplate, inviteEmailSubject string) (inviteID int64) {
	body := fmt.Sprintf(`{"args":{"Email":"%s","Roles":"%s","ExpireDatetime":%d,"EmailTemplate":"%s","EmailSubject":"%s"}}`,
		email, initialRoles, expireDatetime, inviteEmailTemplate, inviteEmailSubject)
	return vit.PostWS(ws, "c.sys.InitiateInvitationByEMail", body).NewID()
}

func InitiateJoinWorkspace(vit *vit.VIT, ws *vit.AppWorkspace, inviteID int64, login string, verificationCode string) {
	profile := vit.GetPrincipal(istructs.AppQName_test1_app1, login)
	vit.PostWS(ws, "c.sys.InitiateJoinWorkspace", fmt.Sprintf(`{"args":{"InviteID":%d,"VerificationCode":"%s"}}`, inviteID, verificationCode), coreutils.WithAuthorizeBy(profile.Token))
}

func WaitForInviteState(vit *vit.VIT, ws *vit.AppWorkspace, inviteState int32, inviteID int64) {
	deadline := time.Now().Add(time.Second * 5)
	var entity []interface{}
	for time.Now().Before(deadline) {
		entity = vit.PostWS(ws, "q.sys.Collection", fmt.Sprintf(`
		{"args":{"Schema":"sys.Invite"},
		"elements":[{"fields":["State","sys.ID"]}],
		"filters":[{"expr":"eq","args":{"field":"sys.ID","value":%d}}]}`, inviteID)).SectionRow(0)
		if inviteState == int32(entity[0].(float64)) {
			return
		}
	}
	panic(fmt.Sprintf("invite [%d] is not in required state [%d] it has state [%d]", inviteID, inviteState, int32(entity[0].(float64))))
}

type joinedWorkspaceDesc struct {
	id                    int64
	isActive              bool
	roles                 string
	invitingWorkspaceWSID istructs.WSID
	wsName                string
}

func FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(vit *vit.VIT, invitingWorkspaceWSID istructs.WSID, login string) joinedWorkspaceDesc {
	resp := vit.PostProfile(vit.GetPrincipal(istructs.AppQName_test1_app1, login), "q.sys.Collection", fmt.Sprintf(`
		{"args":{"Schema":"sys.JoinedWorkspace"},
		"elements":[{"fields":[
			"sys.ID",
			"sys.IsActive",
			"Roles",
			"InvitingWorkspaceWSID",
			"WSName"
		]}],
		"filters":[{"expr":"eq","args":{"field":"InvitingWorkspaceWSID","value":%d}}]}`, invitingWorkspaceWSID))
	const wsNameIdx = 4
	return joinedWorkspaceDesc{
		id:                    int64(resp.SectionRow()[0].(float64)),
		isActive:              resp.SectionRow()[1].(bool),
		roles:                 resp.SectionRow()[2].(string),
		invitingWorkspaceWSID: istructs.WSID(resp.SectionRow()[3].(float64)),
		wsName:                resp.SectionRow()[wsNameIdx].(string),
	}
}

func DenyCreateCDocWSKind_Test(t *testing.T, cdocWSKinds []appdef.QName) {
	vit := vit.NewVIT(t, &vit.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	for _, cdocWSkind := range cdocWSKinds {
		t.Run("deny to create manually cdoc.sys."+cdocWSkind.String(), func(t *testing.T) {
			body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "%s"}}]}`, cdocWSkind.String())
			vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect403()).Println()
		})
	}
}
