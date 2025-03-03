/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"log"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/verifier"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_Verifier(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	userPrincipal := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	verificationToken := ""
	verificationCode := ""
	// initiate verification and get the verification token
	body := fmt.Sprintf(`
			{
				"args":{
					"Entity":"%s",
					"Field":"EmailField",
					"Email":"%s",
					"TargetWSID": %d,
					"Language":"fr"
				},
				"elements":[{"fields":["VerificationToken"]}]
			}
		`, it.QNameApp1_TestEmailVerificationDoc, it.TestEmail, userPrincipal.ProfileWSID) // targetWSID - is the workspace we're going to use the verified value at
	// call q.sys.InitiateEmailVerification at user profile to avoid guests
	// call in target app
	resp := vit.PostProfile(userPrincipal, "q.sys.InitiateEmailVerification", body)
	email := vit.CaptureEmail()
	require.Equal([]string{it.TestEmail}, email.To)
	require.Equal("Votre code de vérification", email.Subject)
	require.Equal(it.TestSMTPCfg.GetFrom(), email.From)
	require.Empty(email.CC)
	require.Empty(email.BCC)
	r := regexp.MustCompile(`(?P<code>\d{6})`)
	matches := r.FindStringSubmatch(email.Body)
	verificationCode = matches[0]
	verificationToken = resp.SectionRow()[0].(string)
	log.Println(verificationCode)
	match, _ := regexp.MatchString(`Voici votre code de vérification`, email.Body)
	require.True(match)

	// get the verified value token using the verification token"
	body = fmt.Sprintf(`
		{
			"args":{
				"VerificationToken":"%s",
				"VerificationCode":"%s"
			},
			"elements":[{"fields":["VerifiedValueToken"]}]
		}
		`, verificationToken, verificationCode)
	resp = vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", body)
	verifiedValueToken := resp.SectionRow()[0].(string)
	log.Println(verifiedValueToken)

	t.Run("decode the verified value token and check the verified value", func(t *testing.T) {
		vvp := payloads.VerifiedValuePayload{}
		as, err := vit.BuiltIn(istructs.AppQName_test1_app1)
		require.NoError(err)
		gp, err := as.AppTokens().ValidateToken(verifiedValueToken, &vvp)
		require.NoError(err)
		require.Equal(istructs.AppQName_test1_app1, gp.AppQName)
		require.Equal(verifier.VerifiedValueTokenDuration, gp.Duration)
		require.Equal(vit.Now().UTC(), gp.IssuedAt.UTC())
		require.Equal(it.QNameApp1_TestEmailVerificationDoc, vvp.Entity)
		require.Equal("EmailField", vvp.Field)
		require.Equal(it.TestEmail, vvp.Value)
	})

	t.Run("create a doc providing the token as the value for the verifiable field", func(t *testing.T) {
		body := fmt.Sprintf(`
			{
				"cuds": [
					{
						"fields": {
							"sys.ID": 1,
							"sys.QName": "%s",
							"EmailField": "%s"
						}
					}
				]
			}`, it.QNameApp1_TestEmailVerificationDoc, verifiedValueToken)
		ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
		vit.PostProfile(ws.Owner, "c.sys.CUD", body)
	})

	t.Run("bug: one token could be used in any wsid", func(t *testing.T) {
		body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "%s","EmailField": "%s"}}]}`, it.QNameApp1_TestEmailVerificationDoc, verifiedValueToken)
		ws2 := vit.CreateWorkspace(it.SimpleWSParams("testws"+vit.NextName()), userPrincipal)
		vit.PostProfile(ws2.Owner, "c.sys.CUD", body)
	})

	t.Run("read the actual verified field value - it should be the value decoded from the token", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Schema":"%s"},"elements":[{"fields": ["EmailField"]}]}`, it.QNameApp1_TestEmailVerificationDoc)
		ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
		resp := vit.PostProfile(ws.Owner, "q.sys.Collection", body)
		require.Equal(it.TestEmail, resp.SectionRow()[0])
	})
}

func TestVerifierErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// funcs should be called in the user profile
	userPrincipal := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	verificationToken, verificationCode := InitiateEmailVerification(vit, userPrincipal, it.QNameApp1_TestEmailVerificationDoc,
		"EmailField", it.TestEmail, userPrincipal.ProfileWSID, coreutils.WithAuthorizeBy(userPrincipal.Token))

	t.Run("error 400 on set the raw value instead of verified value token for the verified field", func(t *testing.T) {
		body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "%s","EmailField": "%s"}}]}`, it.QNameApp1_TestEmailVerificationDoc, it.TestEmail)
		vit.PostProfile(userPrincipal, "c.sys.CUD", body, coreutils.Expect400("invalid token")).Println()
	})

	// issue a token for email field
	body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, verificationToken, verificationCode)
	resp := vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", body)
	emailVerifiedValueToken := resp.SectionRow()[0].(string)
	t.Run("error 400 on different verification algorithm", func(t *testing.T) {
		// use the email token for the phone field
		body = fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "%s","PhoneField": "%s"}}]}`, it.QNameApp1_TestEmailVerificationDoc, emailVerifiedValueToken)
		vit.PostProfile(userPrincipal, "c.sys.CUD", body, coreutils.Expect400()).Println()
	})

	// test usage the token in a diffrerent app is senceless because the target doc does not exists in the different app

	t.Run("error 400 issue token for one WSID but use it in different WSID", func(t *testing.T) {
		t.Skip("WSID check is not implemented in istructsmem yet")
		body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, verificationToken, verificationCode)
		resp := vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", body)
		emailVerifiedValueToken = resp.SectionRow()[0].(string)

		body = fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "%s","EmailField": "%s"}}]}`, it.QNameApp1_TestEmailVerificationDoc, emailVerifiedValueToken)
		// dws := vit.DummyWS(istructs.AppQName_test1_app1, ws.WSID+1)
		userPrincipal2 := vit.GetPrincipal(istructs.AppQName_test1_app2, "login")
		vit.PostProfile(userPrincipal2, "c.sys.CUD", body, coreutils.Expect500()).Println()
	})
}

func TestVerificationLimits(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	rateLimitName_InitiateEmailVerification := istructsmem.GetFunctionRateLimitName(verifier.QNameQueryInitiateEmailVerification, istructs.RateLimitKind_byWorkspace)

	vit.MockBuckets(istructs.AppQName_test1_app1, rateLimitName_InitiateEmailVerification, irates.BucketState{
		Period:             time.Minute,
		MaxTokensPerPeriod: 1,
	})

	// funcs should be called in the user profile
	userPrincipal := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
	var token, code string

	testWSID := istructs.WSID(1)

	t.Run("q.sys.InitiateEmailVerification limits", func(t *testing.T) {

		// first q.sys.InitiateEmailVerifications are ok
		InitiateEmailVerification(vit, userPrincipal, it.QNameApp1_TestEmailVerificationDoc, "EmailField", it.TestEmail, testWSID, coreutils.WithAuthorizeBy(userPrincipal.Token))

		// 2nd exceeds the limit -> 429 Too many requests
		body := fmt.Sprintf(`{"args":{"Entity":"%s","Field":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken"]}]}`, it.QNameApp1_TestEmailVerificationDoc, "EmailField", it.TestEmail)
		vit.PostProfile(userPrincipal, "q.sys.InitiateEmailVerification", body, coreutils.Expect429())

		// still able to send to call in antoher profile because the limit is per-profile
		otherPrn := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail2)
		InitiateEmailVerification(vit, otherPrn, it.QNameApp1_TestEmailVerificationDoc, "EmailField", it.TestEmail2, testWSID, coreutils.WithAuthorizeBy(otherPrn.Token))

		// proceed to the next minute -> limits will be reset
		vit.TimeAdd(time.Minute)

		// expect no errors
		token, code = InitiateEmailVerification(vit, userPrincipal, it.QNameApp1_TestEmailVerificationDoc, "EmailField", it.TestEmail, testWSID, coreutils.WithAuthorizeBy(userPrincipal.Token))
	})

	t.Run("q.sys.IssueVerifiedValueToken limits", func(t *testing.T) {
		bodyWrongCode := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code+"1")
		bodyGoodCode := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code)

		for i := 0; i < int(verifier.IssueVerifiedValueToken_MaxAllowed); i++ {
			// first 3 calls per hour with a wrong code are allowed, just "code wrong" error is returned
			vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", bodyWrongCode, coreutils.Expect400())
		}

		// 4th code check with a good code is failed as well because the function call limit is exceeded
		vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", bodyGoodCode, coreutils.Expect429())

		// proceed to the next hour to reset limits
		vit.TimeAdd(verifier.IssueVerifiedValueToken_Period)

		// regenerate token and code because previous ones are expired already
		token, code = InitiateEmailVerification(vit, userPrincipal, it.QNameApp1_TestEmailVerificationDoc, "EmailField", it.TestEmail, testWSID, coreutils.WithAuthorizeBy(userPrincipal.Token))
		bodyGoodCode = fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code)

		// now check that limits are restored and that limits are reset on successful code verification
		for i := 0; i < int(verifier.IssueVerifiedValueToken_MaxAllowed+1); i++ {
			vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", bodyGoodCode)
		}
	})
}

func TestForRegistry(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// funcs should be called in the user profile
	userPrincipal := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	verificationToken, verificationCode := InitiateEmailVerificationFunc(vit, func() *coreutils.FuncResponse {
		body := fmt.Sprintf(`{"args":{"Entity":"%s","Field":"EmailField","Email":"%s","TargetWSID":%d,"ForRegistry":true},"elements":[{"fields":["VerificationToken"]}]}`,
			it.QNameApp1_TestEmailVerificationDoc, it.TestEmail, userPrincipal.ProfileWSID)
		resp := vit.PostProfile(userPrincipal, "q.sys.InitiateEmailVerification", body)
		return resp
	})

	body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ForRegistry":true},"elements":[{"fields":["VerifiedValueToken"]}]}`, verificationToken, verificationCode)
	resp := vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", body)
	verifiedValueToken := resp.SectionRow()[0].(string)

	// just expect no errors on validate token for sys/registry
	vvp := payloads.VerifiedValuePayload{}
	as, err := vit.BuiltIn(istructs.AppQName_sys_registry)
	require.NoError(t, err)
	_, err = as.AppTokens().ValidateToken(verifiedValueToken, &vvp)
	require.NoError(t, err)
}
