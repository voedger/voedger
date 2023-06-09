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
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/verifier"
	coreutils "github.com/voedger/voedger/pkg/utils"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_Verifier(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	userPrincipal := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	verificationToken := ""
	verificationCode := ""
	t.Run("initiate verification and get the verification token", func(t *testing.T) {
		emailCaptor := vit.ExpectEmail()
		body := fmt.Sprintf(`
			{
				"args":{
					"Entity":"%s",
					"Field":"EmailField",
					"Email":"%s",
					"TargetWSID": %d
				},
				"elements":[{"fields":["VerificationToken"]}]
			}
		`, it.QNameTestEmailVerificationDoc, it.TestEmail, userPrincipal.ProfileWSID) // targetWSID - is the workspace we're going to use the verified value at
		// call q.sys.InitiateEmailVerification at user profile to avoid guests
		// call in target app
		resp := vit.PostProfile(userPrincipal, "q.sys.InitiateEmailVerification", body)
		email := emailCaptor.Capture()
		require.Equal([]string{it.TestEmail}, email.To)
		require.Equal(verifier.EmailSubject, email.Subject)
		require.Equal(verifier.EmailFrom, email.From)
		require.Empty(email.CC)
		require.Empty(email.BCC)
		r := regexp.MustCompile(`(?P<code>\d{6})`)
		matches := r.FindStringSubmatch(email.Body)
		verificationCode = matches[0]
		verificationToken = resp.SectionRow()[0].(string)
		log.Println(verificationCode)
	})

	verifiedValueToken := ""
	t.Run("get the verified value token using the verification token", func(t *testing.T) {
		body := fmt.Sprintf(`
		{
			"args":{
				"VerificationToken":"%s",
				"VerificationCode":"%s"
			},
			"elements":[{"fields":["VerifiedValueToken"]}]
		}
		`, verificationToken, verificationCode)
		resp := vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", body)
		verifiedValueToken = resp.SectionRow()[0].(string)
	})

	log.Println(verifiedValueToken)

	t.Run("decode the verified value token and check the verified value", func(t *testing.T) {
		vvp := payloads.VerifiedValuePayload{}
		as, err := vit.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)
		gp, err := as.AppTokens().ValidateToken(verifiedValueToken, &vvp)
		require.NoError(err)
		require.Equal(istructs.AppQName_test1_app1, gp.AppQName)
		require.Equal(verifier.VerifiedValueTokenDuration, gp.Duration)
		require.Equal(vit.Now(), gp.IssuedAt)
		require.Equal(it.QNameTestEmailVerificationDoc, vvp.Entity)
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
			}`, it.QNameTestEmailVerificationDoc, verifiedValueToken)
		ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
		vit.PostWS(ws, "c.sys.CUD", body)
	})

	t.Run("bug: one token could be used in any wsid", func(t *testing.T) {
		body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "%s","EmailField": "%s"}}]}`, it.QNameTestEmailVerificationDoc, verifiedValueToken)
		ws2 := vit.CreateWorkspace(it.WSParams{
			Name:         "testws" + vit.NextName(),
			Kind:         it.QNameTestWSKind,
			ClusterID:    istructs.MainClusterID,
			InitDataJSON: `{"IntFld":42}`, // from config template
		}, userPrincipal)
		vit.PostWS(ws2, "c.sys.CUD", body)
	})

	t.Run("read the actual verified field value - it should be the value decoded from the token", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Schema":"%s"},"elements":[{"fields": ["EmailField"]}]}`, it.QNameTestEmailVerificationDoc)
		ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
		resp := vit.PostWS(ws, "q.sys.Collection", body)
		require.Equal(it.TestEmail, resp.SectionRow()[0])
	})
}

func TestVerifierErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	// funcs should be called in the user profile
	userPrincipal := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
	ws := vit.DummyWS(istructs.AppQName_test1_app1, userPrincipal.ProfileWSID)

	verificationToken, verificationCode := InitiateEmailVerification(vit, userPrincipal, it.QNameTestEmailVerificationDoc,
		"EmailField", it.TestEmail, ws.WSID, coreutils.WithAuthorizeBy(userPrincipal.Token))

	t.Run("error 500 on set the raw value instead of verified value token for the verified field", func(t *testing.T) {
		body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "%s","EmailField": "%s"}}]}`, it.QNameTestEmailVerificationDoc, it.TestEmail)
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect500()).Println()
	})

	emailVerifiedValueToken := ""
	t.Run("error 500 on different verification algorithm", func(t *testing.T) {
		// issue a token for email field
		body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, verificationToken, verificationCode)
		resp := vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", body)
		emailVerifiedValueToken = resp.SectionRow()[0].(string)

		// use the email token for the phone field
		body = fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "%s","PhoneField": "%s"}}]}`, it.QNameTestEmailVerificationDoc, emailVerifiedValueToken)
		vit.PostWS(ws, "c.sys.CUD", body, coreutils.Expect500()).Println()
	})

	t.Run("error 500 on wrong app", func(t *testing.T) {
		body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "%s","EmailField": "%s"}}]}`, it.QNameTestEmailVerificationDoc, emailVerifiedValueToken)
		userPrincipal := vit.GetPrincipal(istructs.AppQName_test1_app2, "login")
		wsApp2 := vit.DummyWS(istructs.AppQName_test1_app2, userPrincipal.ProfileWSID)
		vit.PostWS(wsApp2, "c.sys.CUD", body, coreutils.Expect500()).Println()
	})

	t.Run("error 400 issue token for one WSID but use it in different WSID", func(t *testing.T) {
		t.Skip("WSID check is not implemented in istructsmem yet")
		body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, verificationToken, verificationCode)
		resp := vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", body)
		emailVerifiedValueToken = resp.SectionRow()[0].(string)

		body = fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "%s","EmailField": "%s"}}]}`, it.QNameTestEmailVerificationDoc, emailVerifiedValueToken)
		dws := vit.DummyWS(istructs.AppQName_test1_app1, ws.WSID+1)
		vit.PostWS(dws, "c.sys.CUD", body, coreutils.Expect500()).Println()
	})
}

func TestVerificationLimits(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
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
		InitiateEmailVerification(vit, userPrincipal, it.QNameTestEmailVerificationDoc, "EmailField", it.TestEmail, testWSID, coreutils.WithAuthorizeBy(userPrincipal.Token))

		// 2nd exceeds the limit -> 429 Too many requests
		body := fmt.Sprintf(`{"args":{"Entity":"%s","Field":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken"]}]}`, it.QNameTestEmailVerificationDoc, "EmailField", it.TestEmail)
		vit.PostProfile(userPrincipal, "q.sys.InitiateEmailVerification", body, coreutils.Expect429())

		// still able to send to call in antoher profile because the limit is per-profile
		otherPrn := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail2)
		InitiateEmailVerification(vit, otherPrn, it.QNameTestEmailVerificationDoc, "EmailField", it.TestEmail2, testWSID, coreutils.WithAuthorizeBy(otherPrn.Token))

		// proceed to the next minute -> limits will be reset
		vit.TimeAdd(time.Minute)

		// expect no errors
		token, code = InitiateEmailVerification(vit, userPrincipal, it.QNameTestEmailVerificationDoc, "EmailField", it.TestEmail, testWSID, coreutils.WithAuthorizeBy(userPrincipal.Token))
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
		token, code = InitiateEmailVerification(vit, userPrincipal, it.QNameTestEmailVerificationDoc, "EmailField", it.TestEmail, testWSID, coreutils.WithAuthorizeBy(userPrincipal.Token))
		bodyGoodCode = fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`, token, code)

		// now check that limits are restored and that limits are reset on successful code verification
		for i := 0; i < int(verifier.IssueVerifiedValueToken_MaxAllowed+1); i++ {
			vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", bodyGoodCode)
		}
	})
}

func TestForRegistry(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	// funcs should be called in the user profile
	userPrincipal := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	verificationToken, verificationCode := InitiateEmailVerificationFunc(vit, func() *coreutils.FuncResponse {
		body := fmt.Sprintf(`{"args":{"Entity":"%s","Field":"EmailField","Email":"%s","TargetWSID":%d,"ForRegistry":true},"elements":[{"fields":["VerificationToken"]}]}`,
			it.QNameTestEmailVerificationDoc, it.TestEmail, userPrincipal.ProfileWSID)
		resp := vit.PostProfile(userPrincipal, "q.sys.InitiateEmailVerification", body)
		return resp
	})

	body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ForRegistry":true},"elements":[{"fields":["VerifiedValueToken"]}]}`, verificationToken, verificationCode)
	resp := vit.PostProfile(userPrincipal, "q.sys.IssueVerifiedValueToken", body)
	verifiedValueToken := resp.SectionRow()[0].(string)

	// just expect no errors on validate token for sys/registry
	vvp := payloads.VerifiedValuePayload{}
	as, err := vit.AppStructs(istructs.AppQName_sys_registry)
	require.NoError(t, err)
	_, err = as.AppTokens().ValidateToken(verifiedValueToken, &vvp)
	require.NoError(t, err)
}
