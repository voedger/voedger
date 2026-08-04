/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func signUpLoginWithAlias(t *testing.T, vit *it.VIT, appQName appdef.AppQName, pwd, alias string) it.Login {
	t.Helper()
	login := vit.SignUp(vit.NextName()+"@123.com", pwd, appQName)
	vit.SignIn(login)

	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	initiateSetLoginAlias(t, vit, login, alias, sysRegistryToken)
	waitForLoginAlias(t, vit, login, alias)

	return login
}

func initiateResetPasswordByEmailAndCapture(t *testing.T, vit *it.VIT, appQName appdef.AppQName, wsid istructs.WSID, email string) (token, code string, profileWSID, canonicalPseudoWSID istructs.WSID) {
	t.Helper()
	body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID","CanonicalPseudoWSID"]}]}`, appQName, email)
	resp := vit.PostApp(istructs.AppQName_sys_registry, wsid, "q.registry.InitiateResetPasswordByEmail", body)

	emailMessage := vit.CaptureEmail()
	if len(emailMessage.To) != 1 || emailMessage.To[0] != email {
		t.Fatalf("reset code recipients = %v, want [%s]", emailMessage.To, email)
	}
	code = regexp.MustCompile(`\d{6}`).FindString(emailMessage.Body)
	if code == "" {
		t.Fatalf("reset code was not found in email body %q", emailMessage.Body)
	}

	row := resp.SectionRow()
	token = row[0].(string)
	profileWSID = istructs.WSID(row[1].(float64))
	canonicalPseudoWSID = istructs.WSID(row[2].(float64))
	return
}

func issueVerifiedValueTokenForResetPassword(t *testing.T, vit *it.VIT, appQName appdef.AppQName, wsid istructs.WSID, token, code string, profileWSID istructs.WSID) string {
	t.Helper()
	body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`,
		token, code, profileWSID, appQName)
	resp := vit.PostApp(istructs.AppQName_sys_registry, wsid, "q.registry.IssueVerifiedValueTokenForResetPassword", body)
	return resp.SectionRow()[0].(string)
}

func resetPasswordByEmail(t *testing.T, vit *it.VIT, appQName appdef.AppQName, wsid istructs.WSID, verifiedValueToken, newPwd string) {
	t.Helper()
	body := fmt.Sprintf(`{"args":{"AppName":"%s"},"unloggedArgs":{"Email":"%s","NewPwd":"%s"}}`, appQName, verifiedValueToken, newPwd)
	vit.PostApp(istructs.AppQName_sys_registry, wsid, "c.registry.ResetPasswordByEmail", body)
}

func assertResetPasswordInitiationRejected(t *testing.T, vit *it.VIT, appQName appdef.AppQName, email string) {
	t.Helper()
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, email, istructs.CurrentClusterID())
	body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, appQName, email)
	vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.registry.InitiateResetPasswordByEmail", body, it.Expect400("login does not exist"))
}

func TestResetPasswordLimits(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	verifierRateMaxAllowed, verifierRatePeriod := vit.RatePerPeriod(istructs.AppQName_test1_app1, appdef.NewQName(appdef.SysPackage, "VerifierRate"))

	var (
		profileWSID istructs.WSID
		token       string
		code        string
	)

	t.Run("InitiateResetPasswordByEmail", func(t *testing.T) {
		// deplete the real bucket
		for range verifierRateMaxAllowed {
			_, _ = InitiateEmailVerificationFunc(vit, func() *federation.FuncResponse {
				body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
				return vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body)
			})
		}

		// next call -> limit exceeded
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body, httpu.Expect429())

		// proceed to the next period to restore rates
		vit.TimeAdd(verifierRatePeriod)

		// call again to get actual token and code
		token, code = InitiateEmailVerificationFunc(vit, func() *federation.FuncResponse {
			body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
			resp := vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body)

			// here in test we're actually know the profileWSID. But in the realife we don't. So let's show how it should be got:
			// q.sys.InitiateResetPasswordByEmail returns it
			profileWSID = istructs.WSID(resp.SectionRow()[1].(float64))
			return resp
		})
	})

	t.Run("IssueVerifiedValueTokenForResetPassword", func(t *testing.T) {
		wrongCode := code + "1"
		wrongCodeBody := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, wrongCode, profileWSID,
			istructs.AppQName_test1_app1)

		// deplete the real bucket with wrong code calls
		for range verifierRateMaxAllowed {
			vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", wrongCodeBody, httpu.Expect400())
		}

		// next call with correct code -> 429 anyway because limit is exceeded
		goodCodeBody := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code, profileWSID,
			istructs.AppQName_test1_app1)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", goodCodeBody, httpu.Expect429())

		// proceed to the next period to restore rates
		vit.TimeAdd(verifierRatePeriod)

		// regenerate token and code because previous ones are expired already
		token, code = InitiateEmailVerificationFunc(vit, func() *federation.FuncResponse {
			body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
			return vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body)
		})
		goodCodeBody = fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code, profileWSID,
			istructs.AppQName_test1_app1)

		// expect no errors now
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", goodCodeBody)
	})
}
