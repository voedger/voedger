/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"fmt"
	"testing"
	"time"

	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys/verifier"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_ResetPassword(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()
	loginName := hit.NextName() + "@123.com"
	login := hit.SignUp(loginName, "1", istructs.AppQName_test1_app1)
	hit.SignIn(login)

	profileWSID := istructs.WSID(0)
	token, code := InitiateEmailVerificationFunc(hit, func() *utils.FuncResponse {
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, login.Name)
		resp := hit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.sys.InitiateResetPasswordByEmail", body) // null auth policy

		// here in test we're actually know the profileWSID. But in the realife we don't. So let's show how it should be got
		profileWSID = istructs.WSID(resp.SectionRow()[1].(float64))
		return resp
	})

	// sys/registry/pseudo-profile-wsid/q.sys.IssueVerifiedValueTokenForResetPassword
	body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code, profileWSID,
		istructs.AppQName_test1_app1)
	resp := hit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.sys.IssueVerifiedValueTokenForResetPassword", body) // null auth policy
	verifiedValueToken := resp.SectionRow()[0].(string)

	// sys/registry/pseudo-profile-wsid/c.sys.ResetPasswordByEmail
	newPwd := "newPwd"
	body = fmt.Sprintf(`{"args":{"AppName":"%s"},"unloggedArgs":{"Email":"%s","NewPwd":"%s"}}`, istructs.AppQName_test1_app1, verifiedValueToken, newPwd)
	hit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.sys.ResetPasswordByEmail", body) // null auth policy

	// expect no errors on login with new password
	login.Pwd = newPwd
	hit.SignIn(login)
}

func TestIntiateResetPasswordErrors(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()
	prn := hit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	t.Run("400 bad request on bad appQName", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppName":"wrong app","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, prn.Name)
		hit.PostApp(istructs.AppQName_sys_registry, istructs.FirstBaseUserWSID, "q.sys.InitiateResetPasswordByEmail", body, utils.Expect400()).Println()
	})

	t.Run("403 forbidden (wrong workspace) if called not at AppWS", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
		hit.PostApp(istructs.AppQName_sys_registry, istructs.FirstBaseUserWSID, "q.sys.InitiateResetPasswordByEmail", body, utils.Expect403()).Println()
	})

	t.Run("400 bad request on an unknown login", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"unknown"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1)
		hit.PostApp(istructs.AppQName_sys_registry, utils.GetPseudoWSID("unknown", istructs.MainClusterID), "q.sys.InitiateResetPasswordByEmail", body, utils.Expect400()).Println()
	})
}

func TestIssueResetPasswordTokenErrors(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()
	prn := hit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	t.Run("400 bad request on an unknown login", func(t *testing.T) {
		unknownLogin := "unknown"
		pseudoWSID := utils.GetPseudoWSID(unknownLogin, istructs.MainClusterID)
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, unknownLogin)
		hit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.sys.InitiateResetPasswordByEmail", body, utils.Expect400()).Println()
	})

	profileWSID := istructs.WSID(0)
	token, code := InitiateEmailVerificationFunc(hit, func() *utils.FuncResponse {
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
		resp := hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.sys.InitiateResetPasswordByEmail", body)
		profileWSID = istructs.WSID(resp.SectionRow()[1].(float64))
		return resp
	})

	t.Run("400 bad request on bad appQName", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"wrong app"},"elements":[{"fields":["VerifiedValueToken"]}]}`,
			token, code, profileWSID)
		hit.PostApp(istructs.AppQName_sys_registry, profileWSID, "q.sys.IssueVerifiedValueTokenForResetPassword", body, utils.Expect400()).Println()
	})
}

func TestResetPasswordLimits(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()
	prn := hit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
	var (
		profileWSID istructs.WSID
		token       string
		code        string
	)

	t.Run("InitiateResetPasswordByEmail", func(t *testing.T) {
		// mock rate limits
		rateLimitName_InitiateEmailVerification := istructsmem.GetFunctionRateLimitName(verifier.QNameQueryInitiateEmailVerification, istructs.RateLimitKind_byWorkspace)
		hit.MockBuckets(istructs.AppQName_test1_app1, rateLimitName_InitiateEmailVerification, irates.BucketState{
			Period:             time.Minute,
			MaxTokensPerPeriod: 1,
		})

		// 1st call -> ok, do not store the code
		_, _ = InitiateEmailVerificationFunc(hit, func() *utils.FuncResponse {
			body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
			return hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.sys.InitiateResetPasswordByEmail", body)
		})

		// 2nd call -> limit exceeded
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
		hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.sys.InitiateResetPasswordByEmail", body, utils.Expect429())

		// proceed to the next minute to restore rates
		hit.TimeAdd(time.Minute)

		// call again to get actual token and code
		token, code = InitiateEmailVerificationFunc(hit, func() *utils.FuncResponse {
			body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
			resp := hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.sys.InitiateResetPasswordByEmail", body)

			// here in test we're actually know the profileWSID. But in the realife we don't. So let's show how it should be got:
			// q.sys.InitiateResetPasswordByEmail returns it
			profileWSID = istructs.WSID(resp.SectionRow()[1].(float64))
			return resp
		})
	})

	t.Run("IssueVerifiedValueTokenForResetPassword", func(t *testing.T) {
		// mock rate limits
		rateLimitName_IssueVerifiedValueToken := istructsmem.GetFunctionRateLimitName(verifier.QNameQueryIssueVerifiedValueToken, istructs.RateLimitKind_byWorkspace)
		hit.MockBuckets(istructs.AppQName_test1_app1, rateLimitName_IssueVerifiedValueToken, irates.BucketState{
			Period:             time.Minute,
			MaxTokensPerPeriod: 1,
		})

		wrongCode := code + "1"
		wrongCodeBody := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, wrongCode, profileWSID,
			istructs.AppQName_test1_app1)

		// 1st call with wrong code -> 400 bad request
		hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.sys.IssueVerifiedValueTokenForResetPassword", wrongCodeBody, utils.Expect400())

		// 2nd call with wrong code -> mocked limit exceeded, 429 Too many reuqets
		hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.sys.IssueVerifiedValueTokenForResetPassword", wrongCodeBody, utils.Expect429())

		// next calls with correct code -> 429 anyway
		goodCodeBody := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code, profileWSID,
			istructs.AppQName_test1_app1)
		hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.sys.IssueVerifiedValueTokenForResetPassword", goodCodeBody, utils.Expect429())

		// proceed to the next minute to restore rates
		hit.TimeAdd(time.Minute)

		// expect no errors now
		hit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.sys.IssueVerifiedValueTokenForResetPassword", goodCodeBody)

	})
}
