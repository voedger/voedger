/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/untillpro/airs-bp3/utils"
	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	hit "github.com/voedger/voedger/pkg/vit"
)

const cud = "c.sys.CUD"

func CreateArticle(hit *hit.HIT, ws *hit.AppWorkspace) (articleID int64) {
	//Create category
	categoryName := "Awesome food"
	body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.category","name":"%s"}}]}`, categoryName)
	resp := hit.PostWS(ws, cud, body)
	categoryID := resp.NewID()

	//Create group
	groupName := "Bar"
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.food_group","id_category":%d,"name":"%s","guid":"14bfa627-1da6-46f6-a915-c802bf046c30"}}]}`, categoryID, groupName)
	resp = hit.PostWS(ws, cud, body)
	groupID := resp.NewID()

	//Create department
	departmentName := "Sweet dishes"
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.department","name":"%s","id_food_group":%d,"pc_fix_button":0,"rm_fix_button":0}}]}`, departmentName, groupID)
	resp = hit.PostWS(ws, cud, body)
	departmentID := resp.NewID()

	//Create article
	articleName := "Chocolate cake"
	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.articles","name":"%s","id_departament":%d,"article_manual":0,"article_hash":0,"hideonhold":0,"time_active": 0,"control_active":0}}]}`, articleName, departmentID)
	resp = hit.PostWS(ws, cud, body)
	articleID = resp.NewID()
	return articleID
}
func CreateWaiter(hit *hit.HIT, ws *hit.AppWorkspace) (waiterID int64) {
	body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.untill_users","name":"Homer","user_void":0,"user_training":0}}]}`
	resp := hit.PostWS(ws, cud, body)
	waiterID = resp.NewID()
	return waiterID
}

func InitiateEmailVerification(hit *hit.HIT, prn *hit.Principal, entity appdef.QName, field, email string, targetWSID istructs.WSID, opts ...utils.ReqOptFunc) (token, code string) {
	return InitiateEmailVerificationFunc(hit, func() *utils.FuncResponse {
		body := fmt.Sprintf(`{"args":{"Entity":"%s","Field":"%s","Email":"%s","TargetWSID":%d},"elements":[{"fields":["VerificationToken"]}]}`, entity, field, email, targetWSID)
		return hit.PostApp(prn.AppQName, prn.ProfileWSID, "q.sys.InitiateEmailVerification", body, opts...)
	})
}

func InitiateEmailVerificationFunc(hit *hit.HIT, f func() *utils.FuncResponse) (token, code string) {
	emailCaptor := hit.ExpectEmail()
	resp := f()
	emailMessage := emailCaptor.Capture()
	r := regexp.MustCompile(`(?P<code>\d{6})`)
	matches := r.FindStringSubmatch(emailMessage.Body)
	code = matches[0]
	token = resp.SectionRow()[0].(string)
	return
}

func WaitForIndexOffset(hit *hit.HIT, ws *hit.AppWorkspace, index appdef.QName, offset int64) {
	type entity struct {
		Last int64 `json:"Last"`
	}

	body := fmt.Sprintf(`
			{
				"args":{"Query":"select LastOffset from %s where Year = %d and DayOfYear = %d"},
				"elements":[{"fields":["Result"]}]
			}`, index, hit.Now().Year(), hit.Now().YearDay())

	deadline := time.Now().Add(time.Second)

	for time.Now().Before(deadline) {
		resp := hit.PostWS(ws, "q.sys.SqlQuery", body)
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
