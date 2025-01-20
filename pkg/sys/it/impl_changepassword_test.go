/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package sys_it

import (
	"fmt"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_ChangePassword(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	loginName := vit.NextName()
	login := vit.SignUp(loginName, "1", istructs.AppQName_test1_app1)

	// change the password
	// null auth
	newPwd := "2"
	body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"1","NewPassword":"%s"}}`, loginName, istructs.AppQName_test1_app1, newPwd)
	vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.registry.ChangePassword", body)

	// note: previous tokens are still valid after password change

	// expect no errors on login with new password
	login.Pwd = newPwd
	vit.SignIn(login)
}

func TestChangePasswordErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login") // from VIT config

	t.Run("login not found", func(t *testing.T) {
		unexistingLogin := vit.NextName()
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"1","NewPassword":"2"}}`,
			unexistingLogin, istructs.AppQName_test1_app1)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.registry.ChangePassword", body, coreutils.Expect401())
	})

	t.Run("wrong password", func(t *testing.T) {
		vit.TimeAdd(time.Minute) // proceed to the next minute to avoid 429 too many requests
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"2","NewPassword":"3"}}`,
			prn.Login.Name, istructs.AppQName_test1_app1)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.registry.ChangePassword", body, coreutils.Expect401())
	})

	t.Run("rate limit exceed", func(t *testing.T) {
		vit.TimeAdd(time.Minute) // proceed to the next minute to avoid 429 too many requests

		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"2","NewPassword":"3"}}`,
			prn.Login.Name, istructs.AppQName_test1_app1)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.registry.ChangePassword", body, coreutils.Expect401()) // not 429, wrong password

		// >1 calls per minute -> 429
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.registry.ChangePassword", body, coreutils.Expect429())

		// proceed to the next minute -> able to change the password again
		vit.TimeAdd(time.Minute)
		body = fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"2","NewPassword":"3"}}`,
			prn.Login.Name, istructs.AppQName_test1_app1)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.registry.ChangePassword", body, coreutils.Expect401()) // again not 429, wrong password
	})
}

