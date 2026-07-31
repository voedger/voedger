/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/registry"
	it "github.com/voedger/voedger/pkg/vit"
)

// [~server.users/it.TestAuthn_LoginCreation~impl]
// [~server.devices/it.TestAuthn_LoginCreation~impl]
func TestAuthn_LoginCreation(t *testing.T) {
	authnLoginCreationFeature(t)
}

func TestAuthn_LoginAliasManagement(t *testing.T) {
	authnLoginAlias(t)
	authnLoginAliasCollisionsAndValidation(t)
}

// [~server.authnz/it.TestAuthn_LoginStateVisibility~impl]
func TestAuthn_LoginStateVisibility(t *testing.T) {
	authnLoginStateVisibility(t)
}

func TestAuthn_CanonicalLoginEnablementManagement(t *testing.T) {
	authnCanonicalLoginEnablementManagement(t)
}

func TestAuthn_DisabledCanonicalLoginBehavior(t *testing.T) {
	authnDisabledCanonicalLoginSignIn(t)
	authnDisabledCanonicalLoginResetPassword(t)
}

func TestAuthn_SignInAndProfileReadiness(t *testing.T) {
	authnAPIAuthLogin(t)
	authnProfileReadinessErrors(t)
}

func TestAuthn_PrincipalTokenContract(t *testing.T) {
	authnAPIAuthRefresh(t)
	authnSignInErrors(t)
}

func TestAuthn_PasswordLifecycle(t *testing.T) {
	authnPasswordLifecycleFeature(t)
}

func TestAuthn_ExceptionFlows(t *testing.T) {
	authnDeactivateUserProfile(t)
}

func authnLoginCreationFeature(t *testing.T) {
	authnUsersCreate(t)

	authnCreateDevice(t)

	authnCreateLoginCoverage(t)

	t.Run("authn: scn: Login creation succeeds for a deactivated login name", func(t *testing.T) {
		// Given a login was previously created and is now deactivated
		// When Client creates a login with the same name again
		// Then the response status is "201 Created"
		// And a new login is accepted with a fresh profile workspace
		// And the previously deactivated login is no longer reachable for sign-in or token issue
		authnRecreateDeactivatedLogin(t)
	})
}

func authnPasswordLifecycleFeature(t *testing.T) {
	authnBasicUsageChangePasswordAPIv2(t)

	authnChangePasswordErrorsAPIv2(t)

	t.Run("authn: scn: Client resets password by verified email", func(t *testing.T) {
		// Given a user login exists
		// When Client initiates password reset by email
		// And Client verifies the reset code
		// And Client resets the password with the verified value token
		// Then Client can sign in with the new password
		authnBasicUsageResetPassword(t)
	})

	authnResetPasswordByAlias(t)
	authnInitiateResetPasswordErrors(t)
	authnIssueResetPasswordTokenErrors(t)
}

func authnBasicUsageChangePasswordAPIv2(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	t.Run("authn: scn: Client changes user password", func(t *testing.T) {
		// Given a user login exists
		loginName := vit.NextName()
		login := vit.SignUp(loginName, "1", istructs.AppQName_test1_app1)

		body := fmt.Sprintf(`{
			"login":"%s",
			"oldPassword": "1",
			"newPassword": "2"
		}`, login.Name)
		// When Client changes the password with the current password
		resp := vit.POST("api/v2/apps/test1/app1/users/change-password", body)
		// Then the response status is "200 OK"
		require.Empty(t, resp.Body)

		login.Pwd = "2"
		// And Client can sign in with the new password
		vit.SignIn(login)
	})

	t.Run("passwords with special JSON characters", func(t *testing.T) {
		vit.TimeAdd(time.Minute) // reset rate-limit window before this password change
		specialPwd := `p"a\ss`
		specialLoginName := vit.NextName()
		specialLogin := vit.SignUp(specialLoginName, specialPwd, istructs.AppQName_test1_app1)
		vit.SignIn(specialLogin)

		bodyBytes, err := json.Marshal(map[string]any{
			"login":       specialLogin.Name,
			"oldPassword": specialPwd,
			"newPassword": specialPwd + "x",
		})
		require.NoError(t, err)
		resp := vit.POST("api/v2/apps/test1/app1/users/change-password", string(bodyBytes))
		require.Empty(t, resp.Body)

		specialLogin.Pwd = specialPwd + "x"
		vit.SignIn(specialLogin)
	})
}

func authnChangePasswordErrorsAPIv2(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	t.Run("authn: scn: Password change rejects malformed request", func(t *testing.T) {
		badRequests := []string{
			`{}`,
			`{"login":"abc"}`,
			`{"login":"abc","oldPassword": "1"}`,
			`{"login":"abc","newPassword": "2"}`,
			`{"login":1,"oldPassword": "1","newPassword": "2"}`,
			`{"login":"abc","oldPassword": 1,"newPassword": "2"}`,
			`{"login":"abc","oldPassword": "1","newPassword": 2}`,
		}
		for _, body := range badRequests {
			// When Client changes a password without login, oldPassword, or newPassword
			// Then the response status is "400 Bad Request"
			vit.POST("api/v2/apps/test1/app1/users/change-password", body, httpu.Expect400()).Println()
		}
	})

	t.Run("authn: scn: Password change rejects unknown login or wrong current password: unknown login", func(t *testing.T) {
		unknownLogin := vit.NextName()
		body := fmt.Sprintf(`{"login":"%s","oldPassword": "1","newPassword": "2"}`, unknownLogin)
		// When Client changes a password for an unknown login or with the wrong current password
		// Then the response status is "401 Unauthorized"
		vit.POST("api/v2/apps/test1/app1/users/change-password", body, httpu.Expect401()).Println()
	})

	t.Run("authn: scn: Password change rejects unknown login or wrong current password: wrong current password", func(t *testing.T) {
		vit.TimeAdd(time.Minute)
		login := vit.SignUp(vit.NextName(), "current-password", istructs.AppQName_test1_app1)
		body := fmt.Sprintf(`{"login":"%s","oldPassword":"wrong-password","newPassword":"new-password"}`, login.Name)
		// When Client changes a password for an unknown login or with the wrong current password
		// Then the response status is "401 Unauthorized"
		vit.POST("api/v2/apps/test1/app1/users/change-password", body, httpu.Expect401()).Println()
	})
}

func authnBasicUsageResetPassword(t *testing.T) {
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

func authnResetPasswordByAlias(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	appQName := istructs.AppQName_test1_app1

	t.Run("authn: scn: Client resets password by verified alias email", func(t *testing.T) {
		alias := vit.NextName() + "@123.com"

		// Given User Login "jsmith" has active Login Alias "j.smith@example.com"
		login := signUpLoginWithAlias(t, vit, appQName, "old-pwd", alias)

		aliasPseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, alias, istructs.CurrentClusterID())

		// When Client initiates password reset using Login Alias "j.smith@example.com"
		token, code, profileWSID, canonicalPseudoWSID := initiateResetPasswordByEmailAndCapture(t, vit, appQName, aliasPseudoWSID, alias)
		if canonicalPseudoWSID != login.PseudoProfileWSID {
			t.Fatalf("CanonicalPseudoWSID = %d, want %d", canonicalPseudoWSID, login.PseudoProfileWSID)
		}

		// And Client verifies the reset code sent to "j.smith@example.com"
		verifiedValueToken := issueVerifiedValueTokenForResetPassword(t, vit, appQName, aliasPseudoWSID, token, code, profileWSID)

		newPwd := "new-alias-reset-pwd"

		// And Client resets the password with the verified value token
		resetPasswordByEmail(t, vit, appQName, canonicalPseudoWSID, verifiedValueToken, newPwd)

		// Then Client can sign in as User Login "jsmith" with the new password
		login.Pwd = newPwd
		vit.SignIn(login)
	})

	t.Run("authn: scn: Password reset initiation rejects an inactive alias: replaced", func(t *testing.T) {
		alias := vit.NextName() + "@123.com"
		newAlias := vit.NextName() + "@123.com"

		// | operation             |
		// | replaced              |
		// Given User Login "jsmith" had Login Alias "j.smith@example.com"
		login := signUpLoginWithAlias(t, vit, appQName, "pwd-replaced", alias)

		sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token

		// And System <operation> that Login Alias
		// operation = replaced
		initiateSetLoginAlias(t, vit, login, newAlias, sysRegistryToken)
		waitForLoginAlias(t, vit, login, newAlias)

		// When Client initiates password reset using Login Alias "j.smith@example.com"
		// Then the response status is "400 Bad Request"
		assertResetPasswordInitiationRejected(t, vit, appQName, alias)
	})

	t.Run("authn: scn: Password reset initiation rejects an inactive alias: cleared", func(t *testing.T) {
		alias := vit.NextName() + "@123.com"

		// | operation             |
		// | cleared               |
		// Given User Login "jsmith" had Login Alias "j.smith@example.com"
		login := signUpLoginWithAlias(t, vit, appQName, "pwd-cleared", alias)

		sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token

		// And System <operation> that Login Alias
		// operation = cleared
		initiateSetLoginAlias(t, vit, login, "", sysRegistryToken)
		waitForLoginAlias(t, vit, login, "")

		// When Client initiates password reset using Login Alias "j.smith@example.com"
		// Then the response status is "400 Bad Request"
		assertResetPasswordInitiationRejected(t, vit, appQName, alias)
	})
}

func authnDisabledCanonicalLoginResetPassword(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	appQName := istructs.AppQName_test1_app1
	alias := vit.NextName() + "@alias.example.com"
	login := signUpLoginWithAlias(t, vit, appQName, "old-pwd", alias)
	principal := vit.SignIn(login)
	aliasPseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, alias, istructs.CurrentClusterID())
	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	profileWSID := principal.ProfileWSID
	canonicalPseudoWSID := login.PseudoProfileWSID

	t.Run("authn: scn: Password reset initiated before canonical Login disablement can complete", func(t *testing.T) {
		// Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
		// And Client initiated password reset using canonical Login "jsmith@example.com"
		canonicalToken, canonicalCode, gotProfileWSID, gotCanonicalPseudoWSID := initiateResetPasswordByEmailAndCapture(t, vit, appQName, login.PseudoProfileWSID, login.Name)
		require.Equal(t, profileWSID, gotProfileWSID)
		require.Equal(t, canonicalPseudoWSID, gotCanonicalPseudoWSID)
		// And Client verified the reset code and received a VerifiedValueToken
		canonicalVerifiedValueToken := issueVerifiedValueTokenForResetPassword(t, vit, appQName, canonicalPseudoWSID, canonicalToken, canonicalCode, profileWSID)
		// And System disabled the canonical Login "jsmith@example.com"
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// When Client resets the password with the VerifiedValueToken
		login.Pwd = "canonical-reset-pwd"
		resetPasswordByEmail(t, vit, appQName, canonicalPseudoWSID, canonicalVerifiedValueToken, login.Pwd)
		// Then Client can sign in using active LoginAlias "j.smith@example.com" and the new password
		issuePrincipalToken(t, vit, alias, login.Pwd, appQName)
		// And CanonicalLoginEnablement of User Login "jsmith@example.com" remains Disabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, vit, login), true)
	})

	t.Run("authn: scn: Disabled canonical Login rejects only canonical entry operations: password reset", func(t *testing.T) {
		// | operation                                                                      | status           | public failure                     | observable result                              |
		// | initiates password reset using canonical Login "jsmith@example.com"            | 400 Bad Request  | an unknown Login                   | no password-reset verification code is issued  |
		// Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// When Client <operation>
		// operation = initiates password reset using canonical Login "jsmith@example.com"
		// Then the response status is "<status>"
		// status = 400 Bad Request
		// And the response is the same as for <public failure>
		// public failure = an unknown Login
		// And <observable result>
		// observable result = no password-reset verification code is issued
		assertResetPasswordInitiationRejected(t, vit, appQName, login.Name)
	})

	t.Run("authn: scn: Active LoginAlias password reset is unaffected by canonical Login disablement", func(t *testing.T) {
		// Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
		// And CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// When Client initiates password reset using LoginAlias "j.smith@example.com"
		token, code, aliasProfileWSID, aliasCanonicalPseudoWSID := initiateResetPasswordByEmailAndCapture(t, vit, appQName, aliasPseudoWSID, alias)
		require.Equal(t, profileWSID, aliasProfileWSID)
		require.Equal(t, login.PseudoProfileWSID, aliasCanonicalPseudoWSID)
		// And Client verifies the reset code sent to "j.smith@example.com"
		verifiedValueToken := issueVerifiedValueTokenForResetPassword(t, vit, appQName, aliasPseudoWSID, token, code, aliasProfileWSID)
		login.Pwd = "alias-reset-pwd"
		// And Client resets the password with the VerifiedValueToken
		resetPasswordByEmail(t, vit, appQName, aliasCanonicalPseudoWSID, verifiedValueToken, login.Pwd)
		// Then Client can sign in using LoginAlias "j.smith@example.com" and the new password
		issuePrincipalToken(t, vit, alias, login.Pwd, appQName)
		// And CanonicalLoginEnablement of User Login "jsmith@example.com" remains Disabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, vit, login), true)
	})

	t.Run("authn: scn: Re-enabling canonical Login restores canonical entry operations: password reset", func(t *testing.T) {
		// | operation                                                                     | observable result                            |
		// | initiates password reset using canonical Login "jsmith@example.com"           | a password-reset verification code is issued |
		// Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// When System enables the canonical Login "jsmith@example.com"
		setCanonicalLoginEnablement(t, vit, login, true, sysRegistryToken)
		// And Client <operation>
		// operation = initiates password reset using canonical Login "jsmith@example.com"
		_, _, restoredProfileWSID, restoredCanonicalPseudoWSID := initiateResetPasswordByEmailAndCapture(t, vit, appQName, login.PseudoProfileWSID, login.Name)
		// Then <observable result>
		// observable result = a password-reset verification code is issued
		require.Equal(t, profileWSID, restoredProfileWSID)
		require.Equal(t, canonicalPseudoWSID, restoredCanonicalPseudoWSID)
	})
}

func authnInitiateResetPasswordErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	t.Run("400 bad request on bad appQName", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppName":"wrong app","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, prn.Name)
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body, httpu.Expect400()).Println()
	})

	// note: test "called in non-AppWS" is senceless because now func is taken from the workspace -> 400 bad request + "func does not exist in the workspace" anyway

	t.Run("authn: scn: Password reset initiation rejects unknown login", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"unknown"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1)
		// When Client initiates password reset for an unknown login
		// Then the response status is "400 Bad Request"
		vit.PostApp(istructs.AppQName_sys_registry, coreutils.GetPseudoWSID(istructs.NullWSID, "unknown", istructs.CurrentClusterID()), "q.registry.InitiateResetPasswordByEmail", body, httpu.Expect400()).Println()
	})
}

func authnIssueResetPasswordTokenErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, it.TestEmail)

	profileWSID := istructs.WSID(0)
	token, code := InitiateEmailVerificationFunc(vit, func() *federation.FuncResponse {
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, prn.Name)
		resp := vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body)
		profileWSID = istructs.WSID(resp.SectionRow()[1].(float64))
		return resp
	})

	t.Run("authn: scn: Password reset verification rejects wrong verification code", func(t *testing.T) {
		// Given Client initiated password reset by email
		wrongCodeBody := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`,
			token, code+"1", profileWSID, istructs.AppQName_test1_app1)
		// When Client verifies the reset code with a wrong code
		// Then the response status is "400 Bad Request"
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", wrongCodeBody, httpu.Expect400()).Println()
	})

	t.Run("400 bad request on bad appQName", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"wrong app"},"elements":[{"fields":["VerifiedValueToken"]}]}`,
			token, code, profileWSID)
		// note: was at profileWSID. It does not works since https://github.com/voedger/voedger/issues/1311
		// because sys/registry:profileWSID workspace is not initialized -> call at pseudoProfileWSID
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", body, httpu.Expect400()).Println()
	})
}

func authnUsersCreate(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	login := vit.NextName() + "@123.com"
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())
	appWSID := coreutils.PseudoWSIDToAppWSID(pseudoWSID, istructs.DefaultNumAppWorkspaces)
	p := payloads.VerifiedValuePayload{
		VerificationKind: appdef.VerificationKind_EMail,
		WSID:             appWSID,
		Field:            "Email", // CreateEmailLoginParams.Email
		Value:            login,
		Entity:           appdef.NewQName(registry.RegistryPackage, "CreateEmailLoginParams"),
	}
	// Given Client has a valid verified email token
	verifiedEmailToken, err := vit.ITokens.IssueToken(istructs.AppQName_sys_registry, 10*time.Minute, &p)
	require.NoError(err)

	t.Run("authn: scn: Client creates a user login from a verified email token", func(t *testing.T) {
		body := fmt.Sprintf(`{"verifiedEmailToken": "%s","password": "123","displayName": "%s"}`, verifiedEmailToken, login)
		// When Client creates a user login with display name and password
		resp := vit.POST("api/v2/apps/test1/app1/users", body)
		// Then the response status is "201 Created"
		require.Equal(http.StatusCreated, resp.HTTPResp.StatusCode)
		// And the user login is accepted
		// And the user profile workspace creation is started
		prn := vit.SignIn(it.Login{Name: login, Pwd: "123", AppQName: istructs.AppQName_test1_app1})
		log.Println(prn)
	})

	assertMalformedRequest := func(t *testing.T, body, field string) {
		t.Helper()
		resp := vit.POST("api/v2/apps/test1/app1/users", body, httpu.Expect400())
		require.Equal(http.StatusBadRequest, resp.HTTPResp.StatusCode)
		require.Contains(resp.Body, field)
	}

	t.Run("authn: scn: User login creation rejects malformed request: verifiedEmailToken", func(t *testing.T) {
		// | field              |
		// | verifiedEmailToken |
		// When Client creates a user login without "<field>"
		// field = verifiedEmailToken
		// Then the response status is "400 Bad Request"
		// And the response indicates "<field>" is missing
		// field = verifiedEmailToken
		assertMalformedRequest(t, fmt.Sprintf(`{"password":"123","displayName":"%s"}`, login), "verifiedEmailToken")
	})

	t.Run("authn: scn: User login creation rejects malformed request: displayName", func(t *testing.T) {
		// | field              |
		// | displayName        |
		// When Client creates a user login without "<field>"
		// field = displayName
		// Then the response status is "400 Bad Request"
		// And the response indicates "<field>" is missing
		// field = displayName
		assertMalformedRequest(t, fmt.Sprintf(`{"verifiedEmailToken":"%s","password":"123"}`, verifiedEmailToken), "displayName")
	})

	t.Run("authn: scn: User login creation rejects malformed request: password", func(t *testing.T) {
		// | field              |
		// | password           |
		// When Client creates a user login without "<field>"
		// field = password
		// Then the response status is "400 Bad Request"
		// And the response indicates "<field>" is missing
		// field = password
		assertMalformedRequest(t, fmt.Sprintf(`{"verifiedEmailToken":"%s","displayName":"%s"}`, verifiedEmailToken, login), "password")
	})

	t.Run("authn: scn: User login creation rejects an invalid verified email token", func(t *testing.T) {
		// Given Client has an invalid verified email token
		invalidTokenBody := fmt.Sprintf(`{"verifiedEmailToken":"invalid-token","password":"123","displayName":"%s"}`, login)
		// When Client creates a user login with display name and password
		resp := vit.POST("api/v2/apps/test1/app1/users", invalidTokenBody, httpu.Expect400())
		// Then the response status is "400 Bad Request"
		require.Equal(http.StatusBadRequest, resp.HTTPResp.StatusCode)
		// And the response indicates verifiedEmailToken validation failed
		require.Contains(resp.Body, "verifiedEmailToken")
	})
}

func authnAPIAuthLogin(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	require := require.New(t)
	//appDef, err := vit.AppDef(istructs.AppQName_test1_app1)

	loginName1 := vit.NextName()
	login1 := vit.SignUp(loginName1, "pwd1", istructs.AppQName_test1_app1)

	vit.SignIn(login1)

	signIn := func(t *testing.T) map[string]interface{} {
		t.Helper()
		body := fmt.Sprintf(`{"login": "%s","password": "%s"}`, login1.Name, login1.Pwd)
		resp := vit.POST("api/v2/apps/test1/app1/auth/login", body)
		require.Equal(200, resp.HTTPResp.StatusCode)
		result := make(map[string]interface{})
		err := json.Unmarshal([]byte(resp.Body), &result)
		require.NoError(err)
		return result
	}

	t.Run("authn: scn: Subject signs in after profile workspace is ready: user", func(t *testing.T) {
		// | subject |
		// | user    |
		// Given "<subject>" login exists
		// subject = user
		// And the profile workspace for "<subject>" is ready
		// subject = user
		// When Client signs in with login and password
		result := signIn(t)
		// Then the response contains principalToken, expiresInSeconds, and profileWSID
		require.Equal(3600.0, result["expiresInSeconds"])
		require.Greater(istructs.WSID(result["profileWSID"].(float64)), login1.PseudoProfileWSID)
		require.NotEmpty(result["principalToken"].(string))
	})

	t.Run("authn: scn: Principal token uses default TTL when no custom TTL is requested", func(t *testing.T) {
		// Given a login exists
		// When Client signs in with login and password
		result := signIn(t)
		// Then expiresInSeconds matches the default principal token expiration
		require.Equal(3600.0, result["expiresInSeconds"])
	})

	t.Run("authn: scn: Principal token carries authn identity fields: user", func(t *testing.T) {
		// | subject |
		// | user    |
		// Given "<subject>" login exists
		// subject = user
		// And the profile workspace for "<subject>" is ready
		// subject = user
		// When Client signs in with login and password
		result := signIn(t)
		principalToken := result["principalToken"].(string)
		payload := payloads.PrincipalPayload{}
		_, err := vit.ITokens.ValidateToken(principalToken, &payload)
		require.NoError(err)
		// Then the issued principal token identifies its login (the canonical login), subject kind, and profileWSID
		require.Equal(login1.Name, payload.Login)
		require.Equal(istructs.SubjectKind_User, payload.SubjectKind)
		require.Equal(istructs.WSID(result["profileWSID"].(float64)), payload.ProfileWSID)
	})

	t.Run("Bad request", func(t *testing.T) {
		cases := []struct {
			bodies   []string
			expected []string
		}{
			{
				bodies:   []string{"", "{}"},
				expected: []string{`field is empty`, `Object «registry.IssuePrincipalTokenParams»`, `string-field «Login»`, `validate error code 4`, `string-field «Password»`},
			},
			{
				bodies: []string{
					`{"password": "pwd"}`,
					fmt.Sprintf(`{"UnknownField": "%s","password": "pwd"}`, login1.Name),
				},
				expected: []string{`field is empty`, `Object «registry.IssuePrincipalTokenParams»`, `string-field «Login»`, `validate error code 4`},
			},
			{
				bodies: []string{
					`{"login": "pwd"}`,
					fmt.Sprintf(`{"login": "%s","UnknownField": "pwd"}`, login1.Name),
				},
				expected: []string{`field is empty`, `Object «registry.IssuePrincipalTokenParams»`, `string-field «Password»`, `validate error code 4`},
			},
			{
				bodies: []string{
					`{"login": 42}`,
				},
				expected: []string{`field \"login\" must be a string`, `field type mismatch`},
			},
			{
				bodies: []string{
					`{"password": 42}`,
				},
				expected: []string{`field \"password\" must be a string`, `field type mismatch`},
			},
			{
				bodies: []string{
					fmt.Sprintf(`{"UnknownField": "%s","password": "%s"}`, login1.Name, "badpwd"),
				},
				expected: []string{`field is empty`, `Object «registry.IssuePrincipalTokenParams»`, `string-field «Login»`, `validate error code 4`},
			},
		}
		for _, c := range cases {
			for _, body := range c.bodies {
				t.Run(body, func(t *testing.T) {
					resp := vit.POST("api/v2/apps/test1/app1/auth/login", body, httpu.Expect400())
					require.Contains(resp.Body, `"status":400`)
					for _, expected := range c.expected {
						require.Contains(resp.Body, expected)
					}
				})
			}
		}
	})

	t.Run("authn: scn: Sign-in rejects unknown login or wrong password: wrong password", func(t *testing.T) {
		// When Client signs in with unknown login or wrong password
		body := fmt.Sprintf(`{"login": "%s","password": "%s"}`, login1.Name, "badpwd")
		resp := vit.POST("api/v2/apps/test1/app1/auth/login", body, httpu.Expect401())
		// Then the response status is "401 Unauthorized"
		require.JSONEq(`{"status":401,"message":"login or password is incorrect"}`, resp.Body)
	})

	t.Run("Login with special JSON characters in password", func(t *testing.T) {
		specialLoginName := vit.NextName()
		specialLogin := vit.SignUp(specialLoginName, `p"a\ss`, istructs.AppQName_test1_app1)
		vit.SignIn(specialLogin)
	})

}

// [~server.authnz/it.TestRefresh~impl]

func authnAPIAuthRefresh(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	require := require.New(t)

	loginName1 := vit.NextName()
	login1 := vit.SignUp(loginName1, "pwd1", istructs.AppQName_test1_app1)
	prn1 := vit.SignIn(login1)

	t.Run("authn: scn: Client refreshes a principal token", func(t *testing.T) {
		// Given Client has a valid principal token
		// simulate delay to make the new token be different after referesh
		vit.TimeAdd(time.Minute)
		// When Client refreshes the principal token
		resp := vit.POST("api/v2/apps/test1/app1/auth/refresh", "", httpu.WithAuthorizeBy(prn1.Token))
		require.Equal(200, resp.HTTPResp.StatusCode)
		result := make(map[string]interface{})
		err := json.Unmarshal([]byte(resp.Body), &result)
		require.NoError(err)
		require.Equal(3600.0, result["expiresInSeconds"])
		require.Equal(istructs.WSID(result["profileWSID"].(float64)), prn1.ProfileWSID)
		newToken := result["principalToken"].(string)
		// Then the response contains a new principalToken
		require.NotEmpty(newToken)
		require.NotEqual(newToken, prn1.Token)
		payload := payloads.PrincipalPayload{}
		_, err = vit.ITokens.ValidateToken(newToken, &payload)
		require.NoError(err)
		// And the new principalToken preserves the login (canonical), alias, subject kind, and profileWSID from the input token
		require.Equal(login1.Name, payload.Login)
		require.Empty(payload.Alias)
		require.Equal(istructs.SubjectKind_User, payload.SubjectKind)
		require.Equal(prn1.ProfileWSID, payload.ProfileWSID)
	})

	t.Run("authn: scn: Principal token refresh requires an existing token", func(t *testing.T) {
		// Given Client has no principal token
		// When Client refreshes the principal token
		resp := vit.POST("api/v2/apps/test1/app1/auth/refresh", "", httpu.Expect401())
		// Then the response status is "401 Unauthorized"
		require.JSONEq(`{"status":401,"message":"authorization header is empty"}`, resp.Body)
	})

	t.Run("Old token", func(t *testing.T) {
		vit.TimeAdd(time.Hour * 2)
		resp := vit.POST("api/v2/apps/test1/app1/auth/refresh", "", httpu.WithAuthorizeBy(prn1.Token), httpu.Expect401())
		require.JSONEq(`{"status":401,"message":"token expired"}`, resp.Body)
	})

}

func authnDeactivateUserProfile(t *testing.T) {
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	loginName := vit.NextName() + "@123.com"
	pwd := "1"
	login := vit.SignUp(loginName, pwd, istructs.AppQName_test1_app1)
	prn := vit.SignIn(login)

	cdocLoginID := vit.GetCDocLoginID(login)

	// obtain a valid VerifiedValueToken before deactivation: c.registry.ResetPasswordByEmail
	// requires a verified Email field, so the only way to reach the GetCDocLogin lookup is
	// to hand the command a token issued for the still-active login
	profileWSID := istructs.WSID(0)
	verifyToken, verifyCode := InitiateEmailVerificationFunc(vit, func() *federation.FuncResponse {
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, loginName)
		resp := vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body)
		profileWSID = istructs.WSID(resp.SectionRow()[1].(float64))
		return resp
	})
	body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`,
		verifyToken, verifyCode, profileWSID, istructs.AppQName_test1_app1)
	verifiedValueToken := vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID,
		"q.registry.IssueVerifiedValueTokenForResetPassword", body).SectionRow()[0].(string)

	// Given a login exists but is deactivated
	vit.PostProfile(prn, "c.sys.InitiateDeactivateWorkspace", "{}")
	waitForDeactivate(vit, prn.AppQName, prn.ProfileWSID, loginName)

	t.Run("410 Gone on work in deactivated profile", func(t *testing.T) {
		body := `{"args":{"Schema":"sys.UserProfile"},"elements":[{"fields":["sys.ID"]}]}`
		vit.PostProfile(prn, "q.sys.Collection", body, httpu.Expect410()).Println()
	})

	expectedCDocLoginIDStr := fmt.Sprintf("%d", cdocLoginID)
	expectVerboseLine := func() {
		logCap.EventuallyHasLine("cdoc.registry.Login", "is deactivated, treating as missing login", expectedCDocLoginIDStr)
	}

	t.Run("authn: scn: Sign-in rejects a deactivated login with the same error as a missing login", func(t *testing.T) {
		logCap.Reset()
		body := fmt.Sprintf(`{"args":{"Login":"%s","Password":"%s","AppName":"%s"},"elements":[{"fields":["PrincipalToken"]}]}`,
			loginName, pwd, istructs.AppQName_test1_app1)
		// When Client signs in with that login and password
		// Then the response status is "401 Unauthorized"
		// And the response indicates the login or password is incorrect
		vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.registry.IssuePrincipalToken", body,
			it.Expect401("login or password is incorrect")).Println()
		expectVerboseLine()
	})

	t.Run("c.registry.ChangePassword -> 401", func(t *testing.T) {
		logCap.Reset()
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s"},"unloggedArgs":{"OldPassword":"%s","NewPassword":"new"}}`,
			loginName, istructs.AppQName_test1_app1, pwd)
		vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.registry.ChangePassword", body,
			it.Expect401(fmt.Sprintf("login %s does not exist", loginName))).Println()
		expectVerboseLine()
	})

	t.Run("q.registry.InitiateResetPasswordByEmail -> 400", func(t *testing.T) {
		logCap.Reset()
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`,
			istructs.AppQName_test1_app1, loginName)
		vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.registry.InitiateResetPasswordByEmail", body,
			it.Expect400("login does not exist")).Println()
		expectVerboseLine()
	})

	t.Run("c.registry.ResetPasswordByEmail -> 401", func(t *testing.T) {
		logCap.Reset()
		body := fmt.Sprintf(`{"args":{"AppName":"%s"},"unloggedArgs":{"Email":"%s","NewPwd":"new"}}`,
			istructs.AppQName_test1_app1, verifiedValueToken)
		vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.registry.ResetPasswordByEmail", body,
			it.Expect401(fmt.Sprintf("login %s does not exist", loginName))).Println()
		expectVerboseLine()
	})

	t.Run("c.registry.UpdateGlobalRoles -> 401", func(t *testing.T) {
		logCap.Reset()
		sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
		body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","GlobalRoles":""}}`,
			loginName, istructs.AppQName_test1_app1)
		vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.registry.UpdateGlobalRoles", body,
			httpu.WithAuthorizeBy(sysRegistryToken),
			it.Expect401(fmt.Sprintf("login %s does not exist", loginName))).Println()
		expectVerboseLine()
	})
}

func authnRecreateDeactivatedLogin(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	loginName := vit.NextName() + "@123.com"
	pwd := "1"

	// create
	login := vit.SignUp(loginName, pwd, istructs.AppQName_test1_app1)
	prn := vit.SignIn(login)
	cdocLoginID := vit.GetCDocLoginID(login)

	// deactivate
	vit.PostProfile(prn, "c.sys.InitiateDeactivateWorkspace", "{}")
	waitForDeactivate(vit, prn.AppQName, prn.ProfileWSID, loginName)

	// create again with the same name
	newLogin := vit.SignUp(loginName, pwd, istructs.AppQName_test1_app1)
	newPrn := vit.SignIn(newLogin)
	newCDocLoginID := vit.GetCDocLoginID(newLogin)

	require.NotEqual(cdocLoginID, newCDocLoginID, "view.registry.LoginIdx must be rewritten to a new CDoc<Login>")
	require.NotEqual(prn.ProfileWSID, newPrn.ProfileWSID, "recreated login must resolve to a new profile workspace")
}

func authnCreateLoginCoverage(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	t.Run("wrong AppWSID", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":"login1","AppName":"test1/app1","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":1},"unloggedArgs":{"Password":"password"}}`, istructs.SubjectKind_User)
		crc16 := coreutils.CRC16([]byte("login1")) - 1 // simulate crc16 is calculated wrong
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

	newLogin := vit.SignUp(login, "1", istructs.AppQName_test1_app1)
	// wait for acomplishing the profile init
	vit.SignIn(newLogin)

	t.Run("authn: scn: Login creation rejects an active duplicate login", func(t *testing.T) {
		// Given an active login already exists
		// When Client creates the same login again
		// Then the response status is "409 Conflict"
		vit.SignUp(login, "1", istructs.AppQName_test1_app1, it.WithReqOpt(httpu.Expect409()))
	})

	t.Run("authn: scn: Login creation rejects an invalid login name", func(t *testing.T) {
		// see https://dev.untill.com/projects/#!537026
		wrongLogins := []string{
			"哇",
			"test@tesT.com",
			"test@test.com ",
			" test@test.com",
			" test@test.com ",
			".test@test.com",
			"test@test.com.",
			".test@test.com.",
			"test@test..com",
			"-test@test.com",
			"test@test.com-",
			"-test@test.com",
			"-test@test.com-",
			"sys.test@test.com",
			",",
			"test,foo@test.com",
		}
		for _, wrongLogin := range wrongLogins {
			pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, wrongLogin, istructs.CurrentClusterID())
			body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"%s"}}`,
				wrongLogin, istructs.AppQName_test1_app1.String(), istructs.SubjectKind_User, istructs.CurrentClusterID(), "1")
			// When Client creates a login with an invalid login name
			// Then the response status is "400 Bad Request"
			// And the response indicates incorrect login format
			vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "c.registry.CreateLogin", body,
				it.Expect400("incorrect login format"))
		}
	})

	t.Run("allowed special chars in login", func(t *testing.T) {
		if testing.Short() {
			t.Skip()
		}
		allowedSpecialChars := []string{"!", "#", "$", "%", "&", "'", "*", "+", "-", "/", "=", ".", "?", "^", "_", "{", "|", "}", "~", "@"}
		for _, c := range allowedSpecialChars {
			goodLogin := vit.NextName() + c + "x"
			login := vit.SignUp(goodLogin, "1", istructs.AppQName_test1_app1)
			vit.SignIn(login)
		}
	})
}

func authnSignInErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	login := vit.NextName()
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())

	t.Run("authn: scn: Sign-in rejects unknown login or wrong password: unknown login", func(t *testing.T) {
		// When Client signs in with unknown login or wrong password
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "1","AppName": "%s"},"elements":[{"fields":["PrincipalToken", "WSID", "WSError"]}]}`,
			login, istructs.AppQName_test1_app1.String())
		// Then the response status is "401 Unauthorized"
		vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.registry.IssuePrincipalToken", body, httpu.Expect401()).Println()
	})

	newLogin := vit.SignUp(login, "1", istructs.AppQName_test1_app1)
	// wait for acomplishing the profile init
	vit.SignIn(newLogin)

	t.Run("wrong password", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "wrongPass","AppName": "%s"},"elements":[{"fields":[]}]}`,
			login, istructs.AppQName_test1_app1.String())
		vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.registry.IssuePrincipalToken", body, httpu.Expect401()).Println()
	})

	t.Run("authn: scn: Principal token rejects TTL above the maximum", func(t *testing.T) {
		// Given a login exists
		prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "%s","AppName": "%s", "TTLHours":1000},"elements":[{"fields":["PrincipalToken"]}]}`,
			prn.Name, prn.Pwd, prn.AppQName.String())
		// When Client requests a principal token with TTL above the maximum
		// Then the response status is "400 Bad Request"
		// And the response indicates the maximum token TTL
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssuePrincipalToken", body,
			it.Expect400("max token TTL hours is 168 hours"))
	})
}

func authnProfileReadinessErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	appQName := istructs.AppQName_test1_app1

	t.Run("authn: scn: Sign-in reports profile workspace not ready", func(t *testing.T) {
		// Given a login exists
		login := vit.SignUp(vit.NextName(), "pwd-not-ready", appQName)
		vit.SignIn(login)
		// And the profile workspace for the login is not ready
		setLoginProfileState(t, vit, login, istructs.NullWSID, "")
		body := fmt.Sprintf(`{"login":"%s","password":"%s"}`, login.Name, login.Pwd)
		// When Client signs in with login and password
		resp := vit.POST("api/v2/apps/test1/app1/auth/login", body, httpu.Expect409())
		// Then the response status is "409 Conflict"
		require.Equal(t, 409, resp.HTTPResp.StatusCode)
		// And the response indicates the profile workspace is not yet ready
		require.Contains(t, resp.Body, "profile workspace is not yet ready")
	})

	t.Run("authn: scn: Sign-in reports profile workspace creation error", func(t *testing.T) {
		// Given a login exists
		login := vit.SignUp(vit.NextName(), "pwd-create-error", appQName)
		principal := vit.SignIn(login)
		// And profile workspace creation failed for the login
		setLoginProfileState(t, vit, login, principal.ProfileWSID, "profile-create-failed")
		body := fmt.Sprintf(`{"login":"%s","password":"%s"}`, login.Name, login.Pwd)
		// When Client signs in with login and password
		resp := vit.POST("api/v2/apps/test1/app1/auth/login", body, httpu.Expect500())
		// Then the response indicates the profile workspace creation error
		require.Contains(t, resp.Body, "profile-create-failed")
	})
}

func authnLoginAlias(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	appQName := istructs.AppQName_test1_app1
	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
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
		initiateSetLoginAlias(t, vit, login, alias, sysRegistryToken)
		waitForLoginAlias(t, vit, login, alias)
		return login, alias
	}

	t.Run("authn: scn: Alias management rejects caller without System Principal Token", func(t *testing.T) {
		// Given a user login exists
		login := newLogin(t, "pwd-authz")
		// When a caller without a System Principal Token sets a login alias for the user
		// Then the alias change is rejected
		initiateSetLoginAlias(t, vit, login, vit.NextName(), "", httpu.Expect403())
	})

	t.Run("authn: scn: System sets the first Login Alias", func(t *testing.T) {
		// Given a User Login "jsmith" with no Login Alias
		login := newLogin(t, "pwd-first")
		alias := vit.NextName()
		// When System sets the Login Alias "j.smith" for "jsmith"
		initiateSetLoginAlias(t, vit, login, alias, sysRegistryToken)
		waitForLoginAlias(t, vit, login, alias)
		// Then "jsmith" has the active Login Alias "j.smith"
		assertLoginAliasState(t, getLoginCDoc(t, vit, login), alias)
	})

	t.Run("authn: scn: User signs in with original login while alias is active", func(t *testing.T) {
		// Given a user login exists with an active login alias
		login, alias := newLoginWithAlias(t, "pwd-original")
		// And the profile workspace for the user is ready
		// When Client signs in with original login and password
		primaryToken := issuePrincipalToken(t, vit, login.Name, login.Pwd, appQName)
		// Then the response contains principalToken, expiresInSeconds, and profileWSID
		assertPrincipalTokenClaims(t, vit, primaryToken, login.Name, alias)
	})

	t.Run("authn: scn: User signs in with active alias", func(t *testing.T) {
		// Given a user login exists with an active login alias
		login, alias := newLoginWithAlias(t, "pwd-alias")
		// And the profile workspace for the user is ready
		// When Client signs in with alias and password
		aliasToken := issuePrincipalToken(t, vit, alias, login.Pwd, appQName)
		// Then the response contains principalToken, expiresInSeconds, and profileWSID
		assertPrincipalTokenClaims(t, vit, aliasToken, login.Name, alias)
	})

	t.Run("authn: scn: Principal token carries the canonical login and the active alias: alias", func(t *testing.T) {
		// | alias state         | identifier     | alias value      |
		// | has an active alias | alias          | the active alias |
		// Given a user login that <alias state>
		// alias state = has an active alias
		login, alias := newLoginWithAlias(t, "pwd-token-alias")
		// And the profile workspace for the user is ready
		// When Client signs in with <identifier> and password
		// identifier = alias
		token := issuePrincipalToken(t, vit, alias, login.Pwd, appQName)
		// Then the issued principal token's login is the canonical login
		// And its alias is <alias value>
		// alias value = the active alias
		assertPrincipalTokenClaims(t, vit, token, login.Name, alias)
	})

	t.Run("authn: scn: Principal token carries the canonical login and the active alias: original login with active alias", func(t *testing.T) {
		// | alias state         | identifier     | alias value      |
		// | has an active alias | original login | the active alias |
		// Given a user login that <alias state>
		// alias state = has an active alias
		login, alias := newLoginWithAlias(t, "pwd-token-original")
		// And the profile workspace for the user is ready
		// When Client signs in with <identifier> and password
		// identifier = original login
		token := issuePrincipalToken(t, vit, login.Name, login.Pwd, appQName)
		// Then the issued principal token's login is the canonical login
		// And its alias is <alias value>
		// alias value = the active alias
		assertPrincipalTokenClaims(t, vit, token, login.Name, alias)
	})

	t.Run("wrong password through an active alias is rejected", func(t *testing.T) {
		_, alias := newLoginWithAlias(t, "pwd-wrong")
		issuePrincipalToken(t, vit, alias, "wrong-password", appQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("setting the same alias is idempotent", func(t *testing.T) {
		login, alias := newLoginWithAlias(t, "pwd-idempotent")
		initiateSetLoginAlias(t, vit, login, alias, sysRegistryToken)
		waitForLoginAlias(t, vit, login, alias)

		cdocLogin := getLoginCDoc(t, vit, login)
		require.Empty(cdocLogin["AliasError"])
		require.Equal(float64(0), cdocLogin["AliasInProc"])
	})

	t.Run("authn: scn: System replaces an existing Login Alias", func(t *testing.T) {
		// Given a User Login "jsmith" with the active Login Alias "j.smith"
		login, alias1 := newLoginWithAlias(t, "pwd-replace")
		alias2 := vit.NextName()
		// When System sets the Login Alias "john.smith" for "jsmith"
		initiateSetLoginAlias(t, vit, login, alias2, sysRegistryToken)
		waitForLoginAlias(t, vit, login, alias2)
		// Then "jsmith" has the active Login Alias "john.smith"
		assertLoginAliasState(t, getLoginCDoc(t, vit, login), alias2)
		// And "j.smith" is no longer active
		issuePrincipalToken(t, vit, alias1, login.Pwd, appQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: Sign-in rejects a previous alias after alias update", func(t *testing.T) {
		// Given a user login exists
		login, previousAlias := newLoginWithAlias(t, "pwd-previous")
		// And the profile workspace for the user is ready
		// And System updated the user's login alias
		newAlias := vit.NextName()
		initiateSetLoginAlias(t, vit, login, newAlias, sysRegistryToken)
		waitForLoginAlias(t, vit, login, newAlias)
		// When Client signs in with the previous alias and password
		// Then the response status is "401 Unauthorized"
		issuePrincipalToken(t, vit, previousAlias, login.Pwd, appQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: System clears a Login Alias", func(t *testing.T) {
		// Given a User Login "jsmith" with the active Login Alias "j.smith"
		login, _ := newLoginWithAlias(t, "pwd-clear")
		// When System clears the Login Alias for "jsmith"
		initiateSetLoginAlias(t, vit, login, "", sysRegistryToken)
		waitForLoginAlias(t, vit, login, "")
		// Then "jsmith" has no active Login Alias
		assertLoginAliasState(t, getLoginCDoc(t, vit, login), "")
	})

	t.Run("authn: scn: Sign-in rejects a cleared alias", func(t *testing.T) {
		// Given a user login exists
		login, clearedAlias := newLoginWithAlias(t, "pwd-cleared-signin")
		// And the profile workspace for the user is ready
		// And System cleared the user's login alias
		initiateSetLoginAlias(t, vit, login, "", sysRegistryToken)
		waitForLoginAlias(t, vit, login, "")
		// When Client signs in with the cleared alias and password
		// Then the response status is "401 Unauthorized"
		issuePrincipalToken(t, vit, clearedAlias, login.Pwd, appQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: Existing principal token retains login and alias after alias changes", func(t *testing.T) {
		// Given Client has a valid principal token issued while a login alias is active
		login, alias := newLoginWithAlias(t, "pwd-snapshot")
		principal := vit.SignIn(login)
		tokenBeforeClear := issuePrincipalToken(t, vit, alias, login.Pwd, appQName)
		// When System updates or clears that login alias
		initiateSetLoginAlias(t, vit, login, "", sysRegistryToken)
		waitForLoginAlias(t, vit, login, "")
		// Then the existing principal token remains valid until normal expiration
		assertPrincipalTokenClaims(t, vit, tokenBeforeClear, login.Name, alias)

		vit.TimeAdd(time.Minute)
		prnWithAliasSnapshot := &it.Principal{
			Login:       login,
			Token:       tokenBeforeClear,
			ProfileWSID: principal.ProfileWSID,
		}
		body := `{"args":{},"elements":[{"fields":["NewPrincipalToken"]}]}`
		resp := vit.PostProfile(prnWithAliasSnapshot, "q.sys.RefreshPrincipalToken", body)
		refreshedToken := resp.SectionRow()[0].(string)
		require.NotEqual(tokenBeforeClear, refreshedToken)
		// And the existing principal token retains the login (canonical) and alias captured at issue time
		assertPrincipalTokenClaims(t, vit, refreshedToken, login.Name, alias)
	})

	t.Run("authn: scn: Principal token carries the canonical login and the active alias: original login without active alias", func(t *testing.T) {
		// | alias state         | identifier     | alias value      |
		// | has no active alias | original login | empty            |
		// Given a user login that <alias state>
		// alias state = has no active alias
		login := newLogin(t, "pwd-no-alias")
		// And the profile workspace for the user is ready
		// When Client signs in with <identifier> and password
		// identifier = original login
		token := issuePrincipalToken(t, vit, login.Name, login.Pwd, appQName)
		// Then the issued principal token's login is the canonical login
		// And its alias is <alias value>
		// alias value = empty
		assertPrincipalTokenClaims(t, vit, token, login.Name, "")
	})

	t.Run("clearing when no alias is set is idempotent", func(t *testing.T) {
		login := newLogin(t, "pwd-clear-empty")
		initiateSetLoginAlias(t, vit, login, "", sysRegistryToken)
		waitForLoginAlias(t, vit, login, "")

		cdocLogin := getLoginCDoc(t, vit, login)
		require.Empty(cdocLogin["AliasError"])
		require.Equal(float64(0), cdocLogin["AliasInProc"])
	})

	t.Run("cleared alias can be reused by another login", func(t *testing.T) {
		login, clearedAlias := newLoginWithAlias(t, "pwd-original-owner")
		initiateSetLoginAlias(t, vit, login, "", sysRegistryToken)
		waitForLoginAlias(t, vit, login, "")
		reuseLogin := newLogin(t, "pwd-reuse")

		initiateSetLoginAlias(t, vit, reuseLogin, clearedAlias, sysRegistryToken)
		waitForLoginAlias(t, vit, reuseLogin, clearedAlias)

		issuePrincipalToken(t, vit, clearedAlias, login.Pwd, appQName, it.Expect401("login or password is incorrect"))
		aliasToken := issuePrincipalToken(t, vit, clearedAlias, reuseLogin.Pwd, appQName)
		assertPrincipalTokenClaims(t, vit, aliasToken, reuseLogin.Name, clearedAlias)
	})
}

func authnLoginAliasCollisionsAndValidation(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	appQName := istructs.AppQName_test1_app1
	login1 := vit.SignUp(vit.NextName(), "pwd1", appQName)
	login2 := vit.SignUp(vit.NextName(), "pwd2", appQName)
	vit.SignIn(login1)
	vit.SignIn(login2)

	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	alias1 := vit.NextName()
	initiateSetLoginAlias(t, vit, login1, alias1, sysRegistryToken)
	waitForLoginAlias(t, vit, login1, alias1)

	t.Run("authn: scn: Alias creation rejects an invalid sign-in identifier", func(t *testing.T) {
		// Given a user login exists
		invalidAliasLogin := vit.SignUp(vit.NextName(), "pwd3", appQName)
		vit.SignIn(invalidAliasLogin)
		// When System sets an invalid login alias for the user
		// Then the alias change is rejected
		// And the response indicates incorrect login format
		initiateSetLoginAlias(t, vit, invalidAliasLogin, "test@test..com", sysRegistryToken, it.Expect400("incorrect login format"))
	})

	t.Run("authn: scn: Alias creation or update rejects a colliding identifier: creates login", func(t *testing.T) {
		// | operation | identifier   |
		// | creates   | login        |
		// Given a user login exists
		collisionLogin := vit.SignUp(vit.NextName(), "pwd4", appQName)
		vit.SignIn(collisionLogin)

		// And another "<identifier>" exists in the same application
		// identifier = login
		// When System "<operation>" the user's login alias using that value
		// operation = creates
		initiateSetLoginAlias(t, vit, collisionLogin, login2.Name, sysRegistryToken)
		// Then the alias change is rejected as a conflict
		waitForLoginAliasError(t, vit, collisionLogin)
		issuePrincipalToken(t, vit, login2.Name, login2.Pwd, appQName)
		issuePrincipalToken(t, vit, login2.Name, collisionLogin.Pwd, appQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: Alias creation or update rejects a colliding identifier: creates active alias", func(t *testing.T) {
		// | operation | identifier   |
		// | creates   | active alias |
		// Given a user login exists
		collisionLogin := vit.SignUp(vit.NextName(), "pwd5", appQName)
		vit.SignIn(collisionLogin)

		// And another "<identifier>" exists in the same application
		// identifier = active alias
		// When System "<operation>" the user's login alias using that value
		// operation = creates
		initiateSetLoginAlias(t, vit, collisionLogin, alias1, sysRegistryToken)
		// Then the alias change is rejected as a conflict
		waitForLoginAliasError(t, vit, collisionLogin)
		issuePrincipalToken(t, vit, alias1, login1.Pwd, appQName)
		issuePrincipalToken(t, vit, alias1, collisionLogin.Pwd, appQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: Alias creation or update rejects a colliding identifier: updates login", func(t *testing.T) {
		// | operation | identifier   |
		// | updates   | login        |
		// Given a user login exists
		collisionLogin := vit.SignUp(vit.NextName(), "pwd-update-login", appQName)
		vit.SignIn(collisionLogin)
		initialAlias := vit.NextName()
		initiateSetLoginAlias(t, vit, collisionLogin, initialAlias, sysRegistryToken)
		waitForLoginAlias(t, vit, collisionLogin, initialAlias)

		// And another "<identifier>" exists in the same application
		// identifier = login
		// When System "<operation>" the user's login alias using that value
		// operation = updates
		initiateSetLoginAlias(t, vit, collisionLogin, login2.Name, sysRegistryToken)
		// Then the alias change is rejected as a conflict
		waitForLoginAliasError(t, vit, collisionLogin)
	})

	t.Run("authn: scn: Alias creation or update rejects a colliding identifier: updates active alias", func(t *testing.T) {
		// | operation | identifier   |
		// | updates   | active alias |
		// Given a user login exists
		collisionLogin := vit.SignUp(vit.NextName(), "pwd-update-alias", appQName)
		vit.SignIn(collisionLogin)
		initialAlias := vit.NextName()
		initiateSetLoginAlias(t, vit, collisionLogin, initialAlias, sysRegistryToken)
		waitForLoginAlias(t, vit, collisionLogin, initialAlias)

		// And another "<identifier>" exists in the same application
		// identifier = active alias
		// When System "<operation>" the user's login alias using that value
		// operation = updates
		initiateSetLoginAlias(t, vit, collisionLogin, alias1, sysRegistryToken)
		// Then the alias change is rejected as a conflict
		waitForLoginAliasError(t, vit, collisionLogin)
	})

	t.Run("authn: scn: Login creation rejects an existing active alias", func(t *testing.T) {
		// Given a login exists with an active login alias
		// When Client creates a login using that alias value
		// Then the response status is "409 Conflict"
		vit.SignUp(alias1, "pwd6", appQName, it.WithReqOpt(httpu.Expect409()))
	})
}

func authnCreateDevice(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	t.Run("authn: scn: Client creates a device login", func(t *testing.T) {
		// When Client creates a device login for an application
		deviceLogin := vit.SignUpDevice(istructs.AppQName_test1_app2)
		// Then the response status is "201 Created"
		// And the response contains generated device login and password
		require.NotEmpty(deviceLogin.Name)
		require.NotEmpty(deviceLogin.Pwd)
		// And the device profile workspace creation is started
		log.Println(deviceLogin.Name)
	})

	t.Run("authn: scn: Subject signs in after profile workspace is ready: device", func(t *testing.T) {
		// | subject |
		// | device  |
		// Given "<subject>" login exists
		// subject = device
		deviceLogin := vit.SignUpDevice(istructs.AppQName_test1_app2)
		// And the profile workspace for "<subject>" is ready
		// subject = device
		// When Client signs in with login and password
		devicePrn := vit.SignIn(deviceLogin)
		// Then the response contains principalToken, expiresInSeconds, and profileWSID
		require.NotEmpty(devicePrn.Token)
		require.NotZero(devicePrn.ProfileWSID)
	})

	t.Run("authn: scn: Principal token carries authn identity fields: device", func(t *testing.T) {
		// | subject |
		// | device  |
		// Given "<subject>" login exists
		// subject = device
		deviceLogin := vit.SignUpDevice(istructs.AppQName_test1_app2)
		// And the profile workspace for "<subject>" is ready
		// subject = device
		// When Client signs in with login and password
		devicePrn := vit.SignIn(deviceLogin)
		as, err := vit.BuiltIn(istructs.AppQName_test1_app2)
		require.NoError(err)
		devicePrnPayload := payloads.PrincipalPayload{}
		_, err = as.AppTokens().ValidateToken(devicePrn.Token, &devicePrnPayload)
		require.NoError(err)
		// Then the issued principal token identifies its login (the canonical login), subject kind, and profileWSID
		require.Equal(deviceLogin.Name, devicePrnPayload.Login)
		require.Equal(istructs.SubjectKind_Device, devicePrnPayload.SubjectKind)
		require.Equal(devicePrn.ProfileWSID, devicePrnPayload.ProfileWSID)
	})

	t.Run("exec a simple operation in the device profile", func(t *testing.T) {
		deviceLogin := vit.SignUpDevice(istructs.AppQName_test1_app2)
		devicePrn := vit.SignIn(deviceLogin)
		body := `{"args":{"Schema":"sys.WorkspaceDescriptor"},"elements":[{"fields":["sys.ID"]}]}`
		vit.PostProfile(devicePrn, "q.sys.Collection", body)
	})

	t.Run("refresh the device principal token", func(t *testing.T) {
		deviceLogin := vit.SignUpDevice(istructs.AppQName_test1_app2)
		devicePrn := vit.SignIn(deviceLogin)
		// simulate delay to make the new token be different
		vit.TimeAdd(time.Minute)
		body := `{"args":{},"elements":[{"fields":["NewPrincipalToken"]}]}`
		resp := vit.PostProfile(devicePrn, "q.sys.RefreshPrincipalToken", body)
		require.NotEqual(devicePrn.Token, resp.SectionRow()[0].(string))
	})

	t.Run("authn: scn: Device login creation rejects request body", func(t *testing.T) {
		// When Client creates a device login with a request body
		// Then the response status is "400 Bad Request"
		// And the response indicates unexpected body
		vit.Func(fmt.Sprintf("api/v2/apps/%s/%s/devices", istructs.AppQName_test1_app2.Owner(), istructs.AppQName_test1_app2.Name()), "body",
			httpu.Expect400()).Println()
	})
}

func authnLoginStateVisibility(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	appQName := istructs.AppQName_test1_app1
	// Given User Login "jsmith" exists
	login := vit.SignUp(vit.NextName(), "pwd1", appQName)
	principal := vit.SignIn(login)

	alias := vit.NextName()
	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token

	registryAppStructs, err := vit.IAppStructsProvider.BuiltIn(istructs.AppQName_sys_registry)
	require.NoError(t, err)
	require.Greater(t, registryAppStructs.NumAppWorkspaces(), istructs.NumAppWorkspaces(1))
	targetRegistryWSID := coreutils.PseudoWSIDToAppWSID(login.PseudoProfileWSID, registryAppStructs.NumAppWorkspaces())
	targetAppWSOffset := targetRegistryWSID.BaseWSID() - istructs.FirstBaseAppWSID
	otherAppWSOffset := (targetAppWSOffset + 1) % istructs.WSID(registryAppStructs.NumAppWorkspaces())
	otherRegistryWSID := istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID+otherAppWSOffset)
	targetRegistryOwnerToken := issueRegistryPrincipalToken(t, vit, "target-registry-owner", targetRegistryWSID)

	assertFullLoginState := func(t *testing.T, cdocLogin map[string]any) {
		t.Helper()
		require.Contains(t, cdocLogin, "PwdHash")
		require.Contains(t, cdocLogin, "LoginHash")
		require.NotContains(t, cdocLogin, "CanonicalLoginDisabled")
	}

	t.Run("authn: scn: Reading a Login CDoc is limited to System and the target registry WorkspaceOwner: System", func(t *testing.T) {
		// | caller                                             | result      |
		// | System                                             | succeeds    |
		// When <caller> reads the Login CDoc of User Login "jsmith"
		// caller = System
		cdocLogin := getLoginCDocWithToken(t, vit, login, sysRegistryToken)
		// Then the read <result>
		// result = succeeds
		assertFullLoginState(t, cdocLogin)
	})

	t.Run("authn: scn: Reading a Login CDoc is limited to System and the target registry WorkspaceOwner: target WorkspaceOwner", func(t *testing.T) {
		// | caller                                             | result      |
		// | a WorkspaceOwner of the target registry workspace | succeeds    |
		// When <caller> reads the Login CDoc of User Login "jsmith"
		// caller = a WorkspaceOwner of the target registry workspace
		cdocLogin := getLoginCDocWithToken(t, vit, login, targetRegistryOwnerToken)
		// Then the read <result>
		// result = succeeds
		assertFullLoginState(t, cdocLogin)
	})

	t.Run("authn: scn: Reading a Login CDoc is limited to System and the target registry WorkspaceOwner: other WorkspaceOwner", func(t *testing.T) {
		// | caller                                             | result      |
		// | a WorkspaceOwner of another workspace             | is rejected |
		token := issueRegistryPrincipalToken(t, vit, "other-registry-owner", otherRegistryWSID)
		// When <caller> reads the Login CDoc of User Login "jsmith"
		// caller = a WorkspaceOwner of another workspace
		// Then the read <result>
		// result = is rejected
		getLoginCDocWithToken(t, vit, login, token, httpu.Expect403())
	})

	t.Run("authn: scn: Reading a Login CDoc is limited to System and the target registry WorkspaceOwner: neither authorization", func(t *testing.T) {
		// | caller                                             | result      |
		// | a caller with neither authorization                | is rejected |
		token := issueRegistryPrincipalToken(t, vit, principal.Name, principal.ProfileWSID)
		// When <caller> reads the Login CDoc of User Login "jsmith"
		// caller = a caller with neither authorization
		// Then the read <result>
		// result = is rejected
		getLoginCDocWithToken(t, vit, login, token, httpu.Expect403())
	})

	t.Run("authn: scn: A target registry WorkspaceOwner read returns Login state", func(t *testing.T) {
		// Given User Login "jsmith" has active LoginAlias "j.smith", no alias change in progress, and no alias error
		initiateSetLoginAlias(t, vit, login, alias, sysRegistryToken)
		waitForLoginAlias(t, vit, login, alias)
		// And CanonicalLoginEnablement of User Login "jsmith" is Disabled
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// When a WorkspaceOwner of the target registry workspace reads the Login CDoc of User Login "jsmith"
		cdocLogin := getLoginCDocWithToken(t, vit, login, targetRegistryOwnerToken)
		// Then the Login CDoc indicates CanonicalLoginEnablement is Disabled
		assertStoredCanonicalLoginDisabled(t, cdocLogin, true)
		// And Alias is "j.smith"
		// And AliasInProc is 0
		// And AliasError is empty
		assertLoginAliasState(t, cdocLogin, alias)
	})
}

func authnCanonicalLoginEnablementManagement(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	appQName := istructs.AppQName_test1_app1
	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	newLogin := func(t *testing.T) it.Login {
		t.Helper()
		login := vit.SignUp(vit.NextName(), "pwd-enable", appQName)
		vit.SignIn(login)
		return login
	}
	caller := newLogin(t)
	callerPrincipal := vit.SignIn(caller)
	callerToken := issueRegistryPrincipalToken(t, vit, callerPrincipal.Name, callerPrincipal.ProfileWSID)

	t.Run("existing Login without stored state is enabled", func(t *testing.T) {
		login := newLogin(t)
		cdocLogin := getLoginCDoc(t, vit, login)
		require.NotContains(t, cdocLogin, "CanonicalLoginDisabled")
		issuePrincipalToken(t, vit, login.Name, login.Pwd, appQName)
	})

	t.Run("authn: scn: Canonical Login enablement management requires a System PrincipalToken: disables", func(t *testing.T) {
		// | operation |
		// | disables  |
		// Given CanonicalLoginEnablement of User Login "jsmith" is Enabled
		login := newLogin(t)
		// When a caller without a System PrincipalToken <operation> the canonical Login "jsmith"
		// operation = disables
		setCanonicalLoginEnablement(t, vit, login, false, callerToken, httpu.Expect403())
		// Then the enablement operation is rejected
		// And CanonicalLoginEnablement of User Login "jsmith" remains Enabled
		require.NotContains(t, getLoginCDoc(t, vit, login), "CanonicalLoginDisabled")
	})

	t.Run("authn: scn: Canonical Login enablement management requires a System PrincipalToken: enables", func(t *testing.T) {
		// | operation |
		// | enables   |
		// Given CanonicalLoginEnablement of User Login "jsmith" is Enabled
		login := newLogin(t)
		// When a caller without a System PrincipalToken <operation> the canonical Login "jsmith"
		// operation = enables
		setCanonicalLoginEnablement(t, vit, login, true, callerToken, httpu.Expect403())
		// Then the enablement operation is rejected
		// And CanonicalLoginEnablement of User Login "jsmith" remains Enabled
		require.NotContains(t, getLoginCDoc(t, vit, login), "CanonicalLoginDisabled")
	})

	t.Run("authn: scn: System sets CanonicalLoginEnablement idempotently: Enabled disables Disabled", func(t *testing.T) {
		// | initial state | operation | resulting state |
		// | Enabled       | disables  | Disabled        |
		// Given CanonicalLoginEnablement of User Login "jsmith" is <initial state>
		// initial state = Enabled
		login := newLogin(t)
		// When System <operation> the canonical Login "jsmith" twice
		// operation = disables
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// Then System reads CanonicalLoginEnablement of User Login "jsmith" as "<resulting state>"
		// resulting state = Disabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, vit, login), true)
	})

	t.Run("authn: scn: System sets CanonicalLoginEnablement idempotently: Disabled disables Disabled", func(t *testing.T) {
		// | initial state | operation | resulting state |
		// | Disabled      | disables  | Disabled        |
		// Given CanonicalLoginEnablement of User Login "jsmith" is <initial state>
		// initial state = Disabled
		login := newLogin(t)
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// When System <operation> the canonical Login "jsmith" twice
		// operation = disables
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// Then System reads CanonicalLoginEnablement of User Login "jsmith" as "<resulting state>"
		// resulting state = Disabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, vit, login), true)
	})

	t.Run("authn: scn: System sets CanonicalLoginEnablement idempotently: Disabled enables Enabled", func(t *testing.T) {
		// | initial state | operation | resulting state |
		// | Disabled      | enables   | Enabled         |
		// Given CanonicalLoginEnablement of User Login "jsmith" is <initial state>
		// initial state = Disabled
		login := newLogin(t)
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// When System <operation> the canonical Login "jsmith" twice
		// operation = enables
		setCanonicalLoginEnablement(t, vit, login, true, sysRegistryToken)
		setCanonicalLoginEnablement(t, vit, login, true, sysRegistryToken)
		// Then System reads CanonicalLoginEnablement of User Login "jsmith" as "<resulting state>"
		// resulting state = Enabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, vit, login), false)
		issuePrincipalToken(t, vit, login.Name, login.Pwd, appQName)
	})

	t.Run("authn: scn: System sets CanonicalLoginEnablement idempotently: Enabled enables Enabled", func(t *testing.T) {
		// | initial state | operation | resulting state |
		// | Enabled       | enables   | Enabled         |
		// Given CanonicalLoginEnablement of User Login "jsmith" is <initial state>
		// initial state = Enabled
		login := newLogin(t)
		// When System <operation> the canonical Login "jsmith" twice
		// operation = enables
		setCanonicalLoginEnablement(t, vit, login, true, sysRegistryToken)
		setCanonicalLoginEnablement(t, vit, login, true, sysRegistryToken)
		// Then System reads CanonicalLoginEnablement of User Login "jsmith" as "<resulting state>"
		// resulting state = Enabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, vit, login), false)
	})
}

func authnDisabledCanonicalLoginSignIn(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	appQName := istructs.AppQName_test1_app1
	alias := vit.NextName() + "@example.com"
	login := signUpLoginWithAlias(t, vit, appQName, "pwd1", alias)
	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token

	t.Run("authn: scn: Disabling canonical Login preserves its active LoginAlias", func(t *testing.T) {
		// Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
		// When System disables the canonical Login "jsmith@example.com"
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		cdocLogin := getLoginCDoc(t, vit, login)
		// Then CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		assertStoredCanonicalLoginDisabled(t, cdocLogin, true)
		// And LoginAlias "j.smith@example.com" remains active for User Login "jsmith@example.com"
		assertLoginAliasState(t, cdocLogin, alias)
	})

	t.Run("authn: scn: Disabled canonical Login rejects only canonical entry operations: sign-in", func(t *testing.T) {
		// | operation                                                                      | status           | public failure                     | observable result                              |
		// | signs in using canonical Login "jsmith@example.com" and the correct password   | 401 Unauthorized | an unknown Login or wrong password | no PrincipalToken is returned                  |
		// Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// When Client <operation>
		// operation = signs in using canonical Login "jsmith@example.com" and the correct password
		// Then the response status is "<status>"
		// status = 401 Unauthorized
		// And the response is the same as for <public failure>
		// public failure = an unknown Login or wrong password
		// And <observable result>
		// observable result = no PrincipalToken is returned
		issuePrincipalToken(t, vit, login.Name, login.Pwd, appQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: Active LoginAlias sign-in is unaffected by canonical Login disablement", func(t *testing.T) {
		// Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
		// And CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// And ProfileWorkspace of User Login "jsmith@example.com" is ready
		// When Client signs in using LoginAlias "j.smith@example.com" and the correct password
		token := issuePrincipalToken(t, vit, alias, login.Pwd, appQName)
		// Then the response contains PrincipalToken, expiresInSeconds, and profileWSID
		assertPrincipalTokenClaims(t, vit, token, login.Name, alias)
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, vit, login), true)
	})

	t.Run("authn: scn: Disabled canonical identifier remains reserved", func(t *testing.T) {
		// Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// When Client creates User Login "jsmith@example.com" again
		// Then the response status is "409 Conflict"
		vit.SignUp(login.Name, "other-pwd", appQName, it.WithReqOpt(httpu.Expect409()))
	})

	t.Run("authn: scn: Re-enabling canonical Login restores canonical entry operations: sign-in", func(t *testing.T) {
		// | operation                                                                     | observable result                            |
		// | signs in using canonical Login "jsmith@example.com" and the existing password | a new PrincipalToken is returned             |
		// Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, vit, login, false, sysRegistryToken)
		// When System enables the canonical Login "jsmith@example.com"
		setCanonicalLoginEnablement(t, vit, login, true, sysRegistryToken)
		// And Client <operation>
		// operation = signs in using canonical Login "jsmith@example.com" and the existing password
		token := issuePrincipalToken(t, vit, login.Name, login.Pwd, appQName)
		// Then <observable result>
		// observable result = a new PrincipalToken is returned
		assertPrincipalTokenClaims(t, vit, token, login.Name, alias)
	})
}
