/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"regexp"
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
	emailCaptor := vit.ExpectEmail()
	resp := f()
	emailMessage := emailCaptor.Capture()
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
