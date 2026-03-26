/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/verifier"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_ResetPassword(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	loginName := vit.NextName() + "@123.com"
	login := vit.SignUp(loginName, "1", istructs.AppQName_test1_app1)
	vit.SignIn(login)

	profileWSID := istructs.WSID(0)
	token, code := InitiateEmailVerificationFunc(vit, func() *federation.FuncResponse {
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, login.Name)
		resp := vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body) // null auth policy

		// here in test we're actually know the profileWSID. But in the realife we don't. So let's show how it should be got
		profileWSID = istructs.WSID(resp.SectionRow()[1].(float64))
		return resp
	})

	// sys/registry/pseudo-profile-wsid/q.registry.IssueVerifiedValueTokenForResetPassword
	body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code, profileWSID,
		istructs.AppQName_test1_app1)
	resp := vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", body) // null auth policy
	verifiedValueToken := resp.SectionRow()[0].(string)

	// sys/registry/pseudo-profile-wsid/c.registry.ResetPasswordByEmail
	newPwd := "newPwd"
	body = fmt.Sprintf(`{"args":{"AppName":"%s"},"unloggedArgs":{"Email":"%s","NewPwd":"%s"}}`, istructs.AppQName_test1_app1, verifiedValueToken, newPwd)
	vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.registry.ResetPasswordByEmail", body) // null auth policy

	// expect no errors on login with new password
	login.Pwd = newPwd
	vit.SignIn(login)
}

func TestIntiateResetPasswordErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	t.Run("400 bad request on bad appQName", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppName":"wrong app","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, prn.Name)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body, httpu.Expect400()).Println()
	})

	// note: test "called in non-AppWS" is senceless because now func is taken from the workspace -> 400 bad request + "func does not exist in the workspace" anyway

	t.Run("400 bad request on an unknown login", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"unknown"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1)
		vit.PostApp(istructs.AppQName_sys_registry, coreutils.GetPseudoWSID(istructs.NullWSID, "unknown", istructs.CurrentClusterID()), "q.registry.InitiateResetPasswordByEmail", body, httpu.Expect400()).Println()
	})
}

func TestIssueResetPasswordTokenErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	t.Run("400 bad request on an unknown login", func(t *testing.T) {
		unknownLogin := "unknown"
		pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, unknownLogin, istructs.CurrentClusterID())
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, unknownLogin)
		vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.registry.InitiateResetPasswordByEmail", body, httpu.Expect400()).Println()
	})

	profileWSID := istructs.WSID(0)
	token, code := InitiateEmailVerificationFunc(vit, func() *federation.FuncResponse {
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
		resp := vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body)
		profileWSID = istructs.WSID(resp.SectionRow()[1].(float64))
		return resp
	})

	t.Run("400 bad request on bad appQName", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"wrong app"},"elements":[{"fields":["VerifiedValueToken"]}]}`,
			token, code, profileWSID)
		// note: was at profileWSID. It does not works since https://github.com/voedger/voedger/issues/1311
		// because sys/registry:profileWSID workspace is not initialized -> call at pseudoProfileWSID
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", body, httpu.Expect400()).Println()
	})
}

func TestResetPasswordLimits(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
	var (
		profileWSID istructs.WSID
		token       string
		code        string
	)

	t.Run("InitiateResetPasswordByEmail", func(t *testing.T) {
		// deplete the real bucket (3/hour)
		for range verifier.InitiateEmailVerification_MaxAllowed {
			_, _ = InitiateEmailVerificationFunc(vit, func() *federation.FuncResponse {
				body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
				return vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body)
			})
		}

		// next call -> limit exceeded
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body, httpu.Expect429())

		// proceed to the next period to restore rates
		vit.TimeAdd(verifier.InitiateEmailVerification_Period)

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

		// deplete the real bucket (3/hour) with wrong code calls
		for range verifier.IssueVerifiedValueToken_MaxAllowed {
			vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", wrongCodeBody, httpu.Expect400())
		}

		// next call with correct code -> 429 anyway because limit is exceeded
		goodCodeBody := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code, profileWSID,
			istructs.AppQName_test1_app1)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", goodCodeBody, httpu.Expect429())

		// proceed to the next period to restore rates
		vit.TimeAdd(verifier.IssueVerifiedValueToken_Period)

		// regenerate token and code because previous ones are expired already (VerificationTokenDuration = 10 min < IssueVerifiedValueToken_Period = 1 hour)
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
