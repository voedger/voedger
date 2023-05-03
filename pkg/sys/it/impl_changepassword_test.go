/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package heeus_it

import (
	"fmt"
	"testing"
	"time"

	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_ChangePassword(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	loginName := hit.NextName()
	login := hit.SignUp(loginName, "1", istructs.AppQName_test1_app1)

	// change the password
	// null auth
	newPwd := "2"
	body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"1","NewPassword":"%s"}}`, loginName, istructs.AppQName_test1_app1, newPwd)
	hit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.sys.ChangePassword", body)

	// note: previous tokens are still valid after password change

	// expect no errors on login with new password
	login.Pwd = newPwd
	hit.SignIn(login)
}

func TestChangePasswordErrors(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	prn := hit.GetPrincipal(istructs.AppQName_test1_app1, "login") // from HIT config

	t.Run("login not found", func(t *testing.T) {
		unexistingLogin := hit.NextName()
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"1","NewPassword":"2"}}`,
			unexistingLogin, istructs.AppQName_test1_app1)
		hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.sys.ChangePassword", body, utils.Expect401())
	})

	t.Run("wrong password", func(t *testing.T) {
		hit.TimeAdd(time.Minute) // proceed to the next minute to avoid 429 too many requests
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"2","NewPassword":"3"}}`,
			prn.Login.Name, istructs.AppQName_test1_app1)
		hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.sys.ChangePassword", body, utils.Expect401())
	})

	t.Run("rate limit exceed", func(t *testing.T) {
		hit.TimeAdd(time.Minute) // proceed to the next minute to avoid 429 too many requests

		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"2","NewPassword":"3"}}`,
			prn.Login.Name, istructs.AppQName_test1_app1)
		hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.sys.ChangePassword", body, utils.Expect401()) // not 429, wrong password

		// >1 calls per minute -> 429
		hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.sys.ChangePassword", body, utils.Expect429())

		// proceed to the next minute -> able to change the password again
		hit.TimeAdd(time.Minute)
		body = fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"2","NewPassword":"3"}}`,
			prn.Login.Name, istructs.AppQName_test1_app1)
		hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.sys.ChangePassword", body, utils.Expect401()) // again not 429, wrong password
	})
}
