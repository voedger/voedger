/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestAuthnTechnical_LoginCreation(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	t.Cleanup(vit.TearDown)

	t.Run("wrong AppWSID", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"login1","AppName":"test1/app1","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":1},"unloggedArgs":{"Password":"password"}}`, istructs.SubjectKind_User)
		crc16 := coreutils.CRC16([]byte("login1")) - 1
		pseudoWSID := istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(crc16))
		vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "c.registry.CreateLogin", body,
			it.Expect403("wrong AppWSID: 140737488420870 expected, 140737488420869 got"))
	})

	login := vit.NextName()
	loginPseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())

	t.Run("unknown application", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"my/unknown","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"password"}}`,
			login, istructs.SubjectKind_User, istructs.CurrentClusterID())
		vit.PostApp(istructs.AppQName_sys_registry, loginPseudoWSID, "c.registry.CreateLogin", body, it.Expect400("my/unknown is not found"))
	})

	t.Run("wrong application name", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"wrong-AppName","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":1},"unloggedArgs":{"Password":"different"}}`,
			login, istructs.SubjectKind_User)
		vit.PostApp(istructs.AppQName_sys_registry, loginPseudoWSID, "c.registry.CreateLogin", body,
			it.Expect400("failed to parse app qualified name"))
	})

	t.Run("allowed special chars in login", func(t *testing.T) {
		if testing.Short() {
			t.Skip()
		}
		allowedSpecialChars := []string{"!", "#", "$", "%", "&", "'", "*", "+", "-", "/", "=", ".", "?", "^", "_", "{", "|", "}", "~", "@"}
		for _, char := range allowedSpecialChars {
			goodLogin := vit.NextName() + char + "x"
			createdLogin := vit.SignUp(goodLogin, "1", istructs.AppQName_test1_app1)
			vit.SignIn(createdLogin)
		}
	})
}

func TestAuthnTechnical_PasswordAndLoginTransport(t *testing.T) {
	t.Run("passwords with special JSON characters", func(t *testing.T) {
		vit := it.NewVIT(t, &it.SharedConfig_App1)
		t.Cleanup(vit.TearDown)
		specialPwd := `p"a\ss`
		login := vit.SignUp(vit.NextName(), specialPwd, istructs.AppQName_test1_app1)
		vit.SignIn(login)
		bodyBytes, err := json.Marshal(map[string]any{
			"login":       login.Name,
			"oldPassword": specialPwd,
			"newPassword": specialPwd + "x",
		})
		require.NoError(t, err)
		resp := vit.POST("api/v2/apps/test1/app1/users/change-password", string(bodyBytes))
		require.Empty(t, resp.Body)
		login.Pwd = specialPwd + "x"
		vit.SignIn(login)
	})

	t.Run("login rejects malformed transport", func(t *testing.T) {
		vit := it.NewVIT(t, &it.SharedConfig_App1)
		t.Cleanup(vit.TearDown)
		login := vit.SignUp(vit.NextName(), "pwd-login-transport", istructs.AppQName_test1_app1)
		vit.SignIn(login)
		tests := []struct {
			bodies   []string
			expected []string
		}{
			{
				bodies:   []string{"", "{}"},
				expected: []string{`field is empty`, `Object «registry.IssuePrincipalTokenParams»`, `string-field «Login»`, `validate error code 4`, `string-field «Password»`},
			},
			{
				bodies: []string{
					`{"password":"pwd"}`,
					fmt.Sprintf(`{"UnknownField":"%s","password":"pwd"}`, login.Name),
					fmt.Sprintf(`{"UnknownField":"%s","password":"%s"}`, login.Name, "badpwd"),
				},
				expected: []string{`field is empty`, `Object «registry.IssuePrincipalTokenParams»`, `string-field «Login»`, `validate error code 4`},
			},
			{
				bodies:   []string{`{"login":"pwd"}`, fmt.Sprintf(`{"login":"%s","UnknownField":"pwd"}`, login.Name)},
				expected: []string{`field is empty`, `Object «registry.IssuePrincipalTokenParams»`, `string-field «Password»`, `validate error code 4`},
			},
			{bodies: []string{`{"login":42}`}, expected: []string{`field \"login\" must be a string`, `field type mismatch`}},
			{bodies: []string{`{"password":42}`}, expected: []string{`field \"password\" must be a string`, `field type mismatch`}},
		}
		for _, tc := range tests {
			for _, body := range tc.bodies {
				t.Run(body, func(t *testing.T) {
					resp := vit.POST("api/v2/apps/test1/app1/auth/login", body, httpu.Expect400())
					require.Contains(t, resp.Body, `"status":400`)
					for _, expected := range tc.expected {
						require.Contains(t, resp.Body, expected)
					}
				})
			}
		}
	})

	t.Run("login with special JSON characters in password", func(t *testing.T) {
		vit := it.NewVIT(t, &it.SharedConfig_App1)
		t.Cleanup(vit.TearDown)
		login := vit.SignUp(vit.NextName(), `p"a\ss`, istructs.AppQName_test1_app1)
		vit.SignIn(login)
	})
}

func TestAuthnTechnical_ResetPasswordTransport(t *testing.T) {
	t.Run("initiation rejects malformed app QName", func(t *testing.T) {
		vit := it.NewVIT(t, &it.SharedConfig_App1)
		t.Cleanup(vit.TearDown)
		principal := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
		body := fmt.Sprintf(`{"args":{"AppName":"wrong app","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, principal.Name)
		vit.PostApp(istructs.AppQName_sys_registry, principal.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body, httpu.Expect400()).Println()
	})

	t.Run("verified-token issue rejects malformed app QName", func(t *testing.T) {
		vit := it.NewVIT(t, &it.SharedConfig_App1)
		t.Cleanup(vit.TearDown)
		principal := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)
		profileWSID := istructs.WSID(0)
		token, code := InitiateEmailVerificationFunc(vit, func() *federation.FuncResponse {
			body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, principal.Name)
			resp := vit.PostApp(istructs.AppQName_sys_registry, principal.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body)
			profileWSID = istructs.WSID(resp.SectionRow()[1].(float64))
			return resp
		})
		body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"wrong app"},"elements":[{"fields":["VerifiedValueToken"]}]}`,
			token, code, profileWSID)
		vit.PostApp(istructs.AppQName_sys_registry, principal.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", body, httpu.Expect400()).Println()
	})
}

func TestAuthnTechnical_PrincipalToken(t *testing.T) {
	t.Run("expired token cannot be refreshed", func(t *testing.T) {
		vit := it.NewVIT(t, &it.SharedConfig_App1)
		t.Cleanup(vit.TearDown)
		login := vit.SignUp(vit.NextName(), "pwd-expired-token", istructs.AppQName_test1_app1)
		principal := vit.SignIn(login)
		vit.TimeAdd(2 * time.Hour)
		resp := vit.POST("api/v2/apps/test1/app1/auth/refresh", "", httpu.WithAuthorizeBy(principal.Token), httpu.Expect401())
		require.JSONEq(t, `{"status":401,"message":"token expired"}`, resp.Body)
	})

	t.Run("direct token query rejects wrong password", func(t *testing.T) {
		vit := it.NewVIT(t, &it.SharedConfig_App1)
		t.Cleanup(vit.TearDown)
		login := vit.SignUp(vit.NextName(), "pwd-direct-query", istructs.AppQName_test1_app1)
		vit.SignIn(login)
		body := fmt.Sprintf(`{"args":{"Login":"%s","Password":"wrongPass","AppName":"%s"},"elements":[{"fields":[]}]}`,
			login.Name, login.AppQName)
		vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.registry.IssuePrincipalToken", body, httpu.Expect401()).Println()
	})
}

func TestAuthnTechnical_DeactivatedLoginCommands(t *testing.T) {
	logCapture := logger.StartCapture(t, logger.LogLevelVerbose)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	t.Cleanup(vit.TearDown)
	login := vit.SignUp(vit.NextName()+"@123.com", "pwd-deactivate-technical", istructs.AppQName_test1_app1)
	principal := vit.SignIn(login)
	cdocLoginID := vit.GetCDocLoginID(login)

	profileWSID := istructs.WSID(0)
	verifyToken, verifyCode := InitiateEmailVerificationFunc(vit, func() *federation.FuncResponse {
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, login.AppQName, login.Name)
		resp := vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body)
		profileWSID = istructs.WSID(resp.SectionRow()[1].(float64))
		return resp
	})
	body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`,
		verifyToken, verifyCode, profileWSID, login.AppQName)
	verifiedValueToken := vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID,
		"q.registry.IssueVerifiedValueTokenForResetPassword", body).SectionRow()[0].(string)

	vit.PostProfile(principal, "c.sys.InitiateDeactivateWorkspace", "{}")
	waitForDeactivate(vit, principal.AppQName, principal.ProfileWSID, login.Name)
	expectedCDocLoginID := fmt.Sprintf("%d", cdocLoginID)
	expectMissingLoginLog := func() {
		logCapture.EventuallyHasLine("cdoc.registry.Login", "is deactivated, treating as missing login", expectedCDocLoginID)
	}

	t.Run("work in deactivated profile returns 410", func(t *testing.T) {
		body := `{"args":{"Schema":"sys.UserProfile"},"elements":[{"fields":["sys.ID"]}]}`
		vit.PostProfile(principal, "q.sys.Collection", body, httpu.Expect410()).Println()
	})

	t.Run("ChangePassword treats deactivated login as missing", func(t *testing.T) {
		logCapture.Reset()
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"%s","NewPassword":"new"}}`, login.Name, login.AppQName, login.Pwd)
		vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.registry.ChangePassword", body,
			it.Expect401(fmt.Sprintf("login %s does not exist", login.Name))).Println()
		expectMissingLoginLog()
	})

	t.Run("InitiateResetPasswordByEmail treats deactivated login as missing", func(t *testing.T) {
		logCapture.Reset()
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, login.AppQName, login.Name)
		vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body,
			it.Expect400("login does not exist")).Println()
		expectMissingLoginLog()
	})

	t.Run("ResetPasswordByEmail treats deactivated login as missing", func(t *testing.T) {
		logCapture.Reset()
		body := fmt.Sprintf(`{"args":{"AppName":"%s"},"unloggedArgs":{"Email":"%s","NewPwd":"new"}}`, login.AppQName, verifiedValueToken)
		vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.registry.ResetPasswordByEmail", body,
			it.Expect401(fmt.Sprintf("login %s does not exist", login.Name))).Println()
		expectMissingLoginLog()
	})

	t.Run("UpdateGlobalRoles treats deactivated login as missing", func(t *testing.T) {
		logCapture.Reset()
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","GlobalRoles":""}}`, login.Name, login.AppQName)
		vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.registry.UpdateGlobalRoles", body,
			httpu.WithAuthorizeBy(vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token),
			it.Expect401(fmt.Sprintf("login %s does not exist", login.Name))).Println()
		expectMissingLoginLog()
	})
}

func TestAuthnTechnical_LoginAlias(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	t.Cleanup(vit.TearDown)
	appQName := istructs.AppQName_test1_app1
	systemToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	newLogin := func(t *testing.T, pwd string) it.Login {
		t.Helper()
		login := vit.SignUp(vit.NextName(), pwd, appQName)
		vit.SignIn(login)
		return login
	}
	newLoginWithAlias := func(t *testing.T, pwd string) (it.Login, string) {
		t.Helper()
		login := newLogin(t, pwd)
		alias := vit.NextName()
		initiateSetLoginAlias(t, vit, login, alias, systemToken)
		waitForLoginAlias(t, vit, login, alias)
		return login, alias
	}

	t.Run("wrong password through an active alias is rejected", func(t *testing.T) {
		_, alias := newLoginWithAlias(t, "pwd-alias-wrong")
		issuePrincipalToken(t, vit, alias, "wrong-password", appQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("setting the same alias is idempotent", func(t *testing.T) {
		login, alias := newLoginWithAlias(t, "pwd-alias-idempotent")
		initiateSetLoginAlias(t, vit, login, alias, systemToken)
		waitForLoginAlias(t, vit, login, alias)
		cdocLogin := getLoginCDoc(t, vit, login)
		require.Empty(t, cdocLogin["AliasError"])
		require.Equal(t, float64(0), cdocLogin["AliasInProc"])
	})

	t.Run("clearing when no alias is set is idempotent", func(t *testing.T) {
		login := newLogin(t, "pwd-alias-clear-empty")
		initiateSetLoginAlias(t, vit, login, "", systemToken)
		waitForLoginAlias(t, vit, login, "")
		cdocLogin := getLoginCDoc(t, vit, login)
		require.Empty(t, cdocLogin["AliasError"])
		require.Equal(t, float64(0), cdocLogin["AliasInProc"])
	})

	t.Run("cleared alias can be reused by another login", func(t *testing.T) {
		login, alias := newLoginWithAlias(t, "pwd-original-alias-owner")
		initiateSetLoginAlias(t, vit, login, "", systemToken)
		waitForLoginAlias(t, vit, login, "")
		reuseLogin := newLogin(t, "pwd-alias-reuse")
		initiateSetLoginAlias(t, vit, reuseLogin, alias, systemToken)
		waitForLoginAlias(t, vit, reuseLogin, alias)
		issuePrincipalToken(t, vit, alias, login.Pwd, appQName, it.Expect401("login or password is incorrect"))
		token := issuePrincipalToken(t, vit, alias, reuseLogin.Pwd, appQName)
		assertPrincipalTokenClaims(t, vit, token, reuseLogin.Name, alias)
	})

	t.Run("authn: scn: Deactivated Login identifier can become another Login Alias without exposing profile data", func(t *testing.T) {
		const workspaceName = "shared"

		// Given Profile Workspace of User Login "active@example.com" owns child Workspace "shared" containing value "active"
		activeLogin := vit.SignUp(fmt.Sprintf("active%d@example.com", vit.NextNumber()), "active-password", appQName)
		activePrincipal := vit.SignIn(activeLogin)
		activeWSParams := it.SimpleWSParams(workspaceName)
		activeWSParams.InitDataJSON = `{"IntFld":1,"StrFld":"active"}`
		activeWorkspace := vit.CreateWorkspace(activeWSParams, activePrincipal)

		// And Profile Workspace of User Login "retired@example.com" owns child Workspace "shared" containing value "retired"
		retiredLogin := vit.SignUp(fmt.Sprintf("retired%d@example.com", vit.NextNumber()), "retired-password", appQName)
		retiredPrincipal := vit.SignIn(retiredLogin)
		retiredWSParams := it.SimpleWSParams(workspaceName)
		retiredWSParams.InitDataJSON = `{"IntFld":2,"StrFld":"retired"}`
		retiredWorkspace := vit.CreateWorkspace(retiredWSParams, retiredPrincipal)

		// And Profile Workspace of User Login "retired@example.com" is deactivated
		vit.PostProfile(retiredPrincipal, "c.sys.InitiateDeactivateWorkspace", "{}")
		waitForDeactivate(vit, retiredPrincipal.AppQName, retiredPrincipal.ProfileWSID, retiredLogin.Name)

		// When System sets Login Alias "retired@example.com" for User Login "active@example.com"
		initiateSetLoginAlias(t, vit, activeLogin, retiredLogin.Name, sysRegistryToken)
		waitForLoginAlias(t, vit, activeLogin, retiredLogin.Name)

		// And Client signs in using Login Alias "retired@example.com" and the password of User Login "active@example.com"
		aliasToken := issuePrincipalToken(t, vit, retiredLogin.Name, activeLogin.Pwd, appQName)

		// Then the issued Principal Token identifies User Login "active@example.com" and its Profile Workspace
		payload := payloads.PrincipalPayload{}
		_, err := vit.ITokens.ValidateToken(aliasToken, &payload)
		require.NoError(err)
		require.Equal(activeLogin.Name, payload.Login)
		require.Equal(retiredLogin.Name, payload.Alias)
		require.Equal(activePrincipal.ProfileWSID, payload.ProfileWSID)
		require.NotEqual(retiredPrincipal.ProfileWSID, payload.ProfileWSID)
		aliasPrincipal := &it.Principal{
			Login:       activeLogin,
			Token:       aliasToken,
			ProfileWSID: payload.ProfileWSID,
		}

		// And Client reads value "active" from child Workspace "shared"
		aliasWorkspace := vit.WaitForWorkspace(workspaceName, aliasPrincipal)
		require.Equal(activeWorkspace.WSID, aliasWorkspace.WSID)
		body := `{"args":{"Schema":"app1pkg.test_ws"},"elements":[{"fields":["StrFld"]}]}`
		actualValue := vit.PostWS(aliasWorkspace, "q.sys.Collection", body).SectionRow()[0].(string)
		require.Equal("active", actualValue)

		// But Client does not read value "retired" from child Workspace "shared"
		require.NotEqual(retiredWorkspace.WSID, aliasWorkspace.WSID)
		require.NotEqual("retired", actualValue)
	})
}

func TestAuthnTechnical_DevicePrincipal(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	t.Cleanup(vit.TearDown)

	t.Run("exec a simple operation in the device profile", func(t *testing.T) {
		principal := vit.SignIn(vit.SignUpDevice(istructs.AppQName_test1_app2))
		body := `{"args":{"Schema":"sys.WorkspaceDescriptor"},"elements":[{"fields":["sys.ID"]}]}`
		vit.PostProfile(principal, "q.sys.Collection", body)
	})

	t.Run("refresh the device principal token", func(t *testing.T) {
		principal := vit.SignIn(vit.SignUpDevice(istructs.AppQName_test1_app2))
		vit.TimeAdd(time.Minute)
		body := `{"args":{},"elements":[{"fields":["NewPrincipalToken"]}]}`
		resp := vit.PostProfile(principal, "q.sys.RefreshPrincipalToken", body)
		require.NotEqual(t, principal.Token, resp.SectionRow()[0].(string))
	})
}

func TestAuthnTechnical_DefaultCanonicalLoginState(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	t.Cleanup(vit.TearDown)
	login := vit.SignUp(vit.NextName(), "pwd-default-canonical", istructs.AppQName_test1_app1)
	vit.SignIn(login)
	cdocLogin := getLoginCDoc(t, vit, login)
	require.NotContains(t, cdocLogin, "CanonicalLoginDisabled")
	issuePrincipalToken(t, vit, login.Name, login.Pwd, login.AppQName)
}
