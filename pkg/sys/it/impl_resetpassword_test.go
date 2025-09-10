/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
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
		// mock rate limits
		rateLimitName_InitiateEmailVerification := istructsmem.GetFunctionRateLimitName(verifier.QNameQueryInitiateEmailVerification, istructs.RateLimitKind_byWorkspace)
		vit.MockBuckets(istructs.AppQName_test1_app1, rateLimitName_InitiateEmailVerification, irates.BucketState{
			Period:             time.Minute,
			MaxTokensPerPeriod: 1,
		})

		// 1st call -> ok, do not store the code
		_, _ = InitiateEmailVerificationFunc(vit, func() *federation.FuncResponse {
			body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
			return vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body)
		})

		// 2nd call -> limit exceeded
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body, httpu.Expect429())

		// proceed to the next minute to restore rates
		vit.TimeAdd(time.Minute)

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
		// mock rate limits
		rateLimitName_IssueVerifiedValueToken := istructsmem.GetFunctionRateLimitName(verifier.QNameQueryIssueVerifiedValueToken, istructs.RateLimitKind_byWorkspace)
		vit.MockBuckets(istructs.AppQName_test1_app1, rateLimitName_IssueVerifiedValueToken, irates.BucketState{
			Period:             time.Minute,
			MaxTokensPerPeriod: 1,
		})

		wrongCode := code + "1"
		wrongCodeBody := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, wrongCode, profileWSID,
			istructs.AppQName_test1_app1)

		// 1st call with wrong code -> 400 bad request
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", wrongCodeBody, httpu.Expect400())

		// 2nd call with wrong code -> mocked limit exceeded, 429 Too many reuqets
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", wrongCodeBody, httpu.Expect429())

		// next calls with correct code -> 429 anyway
		goodCodeBody := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code, profileWSID,
			istructs.AppQName_test1_app1)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", goodCodeBody, httpu.Expect429())

		// proceed to the next minute to restore rates
		vit.TimeAdd(time.Minute)

		// expect no errors now
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", goodCodeBody)

	})
}
