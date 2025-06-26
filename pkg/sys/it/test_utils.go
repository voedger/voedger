/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"time"

	"golang.org/x/exp/slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/invite"
	it "github.com/voedger/voedger/pkg/vit"
)

func InitiateEmailVerification(vit *it.VIT, prn *it.Principal, entity appdef.QName, field, email string, targetWSID istructs.WSID, opts ...coreutils.ReqOptFunc) (token, code string) {
	return InitiateEmailVerificationFunc(vit, func() *coreutils.FuncResponse {
		body := fmt.Sprintf(`{"args":{"Entity":"%s","Field":"%s","Email":"%s","TargetWSID":%d},"elements":[{"fields":["VerificationToken"]}]}`, entity, field, email, targetWSID)
		return vit.PostApp(prn.AppQName, prn.ProfileWSID, "q.sys.InitiateEmailVerification", body, opts...)
	})
}

func InitiateEmailVerificationFunc(vit *it.VIT, f func() *coreutils.FuncResponse) (token, code string) {
	resp := f()
	emailMessage := vit.CaptureEmail()
	r := regexp.MustCompile(`(?P<code>\d{6})`)
	matches := r.FindStringSubmatch(emailMessage.Body)
	code = matches[0]
	token = resp.SectionRow()[0].(string)
	return
}

func WaitForIndexOffset(vit *it.VIT, ws *it.AppWorkspace, index appdef.QName, offset istructs.Offset) {
	type entity struct {
		LastOffset istructs.Offset
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
		if e.LastOffset == offset {
			break
		}
	}
}

func InitiateInvitationByEMail(vit *it.VIT, ws *it.AppWorkspace, expireDatetime int64, email, initialRoles, inviteEmailTemplate, inviteEmailSubject string) (inviteID istructs.RecordID) {
	vit.T.Helper()
	body := fmt.Sprintf(`{"args":{"Email":"%s","Roles":"%s","ExpireDatetime":%d,"EmailTemplate":"%s","EmailSubject":"%s"}}`,
		email, initialRoles, expireDatetime, inviteEmailTemplate, inviteEmailSubject)
	return vit.PostWS(ws, "c.sys.InitiateInvitationByEMail", body).NewID()
}

func InitiateJoinWorkspace(vit *it.VIT, ws *it.AppWorkspace, inviteID istructs.RecordID, login *it.Principal, verificationCode string, opts ...coreutils.ReqOptFunc) {
	vit.T.Helper()
	opts = append(opts, coreutils.WithAuthorizeBy(login.Token))
	vit.PostWS(ws, "c.sys.InitiateJoinWorkspace", fmt.Sprintf(`{"args":{"InviteID":%d,"VerificationCode":"%s"}}`, inviteID, verificationCode), opts...)
}

func WaitForInviteState(vit *it.VIT, ws *it.AppWorkspace, inviteID istructs.RecordID, inviteStatesSeq ...invite.State) {
	deadline := it.TestDeadline()
	var actualInviteState invite.State
	for time.Now().Before(deadline) {
		entity := vit.PostWS(ws, "q.sys.Collection", fmt.Sprintf(`
		{"args":{"Schema":"sys.Invite"},
		"elements":[{"fields":["State","sys.ID"]}],
		"filters":[{"expr":"eq","args":{"field":"sys.ID","value":%d}}]}`, inviteID)).SectionRow(0)
		actualInviteState = invite.State(entity[0].(float64))
		if inviteStatesSeq[len(inviteStatesSeq)-1] == actualInviteState {
			return
		}
		if !slices.Contains(inviteStatesSeq, actualInviteState) {
			break
		}
	}
	_, file, line, _ := runtime.Caller(1)
	vit.T.Fatalf("%s:%d: invite %d is failed achieve the state %d. The last state was %d", file, line, inviteID, inviteStatesSeq[len(inviteStatesSeq)-1], actualInviteState)
}

type joinedWorkspaceDesc struct {
	id                    int64
	isActive              bool
	roles                 string
	invitingWorkspaceWSID istructs.WSID
	wsName                string
}

func FindCDocJoinedWorkspaceByInvitingWorkspaceWSIDAndLogin(vit *it.VIT, invitingWorkspaceWSID istructs.WSID, login *it.Principal) joinedWorkspaceDesc {
	vit.T.Helper()
	resp := vit.PostProfile(login, "q.sys.Collection", fmt.Sprintf(`
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
