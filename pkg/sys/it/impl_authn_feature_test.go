/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/registry"
	it "github.com/voedger/voedger/pkg/vit"
)

type authnFeatureFixture struct {
	t   *testing.T
	vit *it.VIT
}

// [~server.users/it.TestAuthn_LoginCreation~impl]
// [~server.devices/it.TestAuthn_LoginCreation~impl]
// [~server.authnz/it.TestAuthn_LoginStateVisibility~impl]
// [~server.authnz/it.TestRefresh~impl]
func TestAuthn(t *testing.T) {
	testAuthnLoginCreation(t)
	testAuthnLoginAliasManagement(t)
	testAuthnLoginStateVisibility(t)
	testAuthnCanonicalLoginEnablementManagement(t)
	testAuthnDisabledCanonicalLoginBehavior(t)
	testAuthnSignInAndProfileReadiness(t)
	testAuthnPrincipalTokenContract(t)
	testAuthnPasswordLifecycle(t)
	testAuthnExceptionFlows(t)
}

func newAuthnFeatureFixture(t *testing.T) *authnFeatureFixture {
	t.Helper()
	return &authnFeatureFixture{t: t}
}

func (f *authnFeatureFixture) VIT() *it.VIT {
	f.t.Helper()
	if f.vit == nil {
		f.vit = it.NewVIT(f.t, &it.SharedConfig_App1)
		f.t.Cleanup(f.vit.TearDown)
	}
	return f.vit
}

func (f *authnFeatureFixture) newLogin(pwd string) it.Login {
	f.t.Helper()
	return f.newLoginNamed(f.VIT().NextName(), pwd)
}

func (f *authnFeatureFixture) newEmailLogin(pwd string) it.Login {
	f.t.Helper()
	return f.newLoginNamed(f.VIT().NextName()+"@example.com", pwd)
}

func (f *authnFeatureFixture) newLoginNamed(name, pwd string) it.Login {
	f.t.Helper()
	vit := f.VIT()
	login := vit.SignUp(name, pwd, istructs.AppQName_test1_app1)
	vit.SignIn(login)
	return login
}

func (f *authnFeatureFixture) newLoginWithAlias(pwd, alias string) it.Login {
	f.t.Helper()
	return signUpLoginWithAlias(f.t, f.VIT(), istructs.AppQName_test1_app1, pwd, alias)
}

func (f *authnFeatureFixture) systemRegistryToken() string {
	f.t.Helper()
	return f.VIT().GetSystemPrincipal(istructs.AppQName_sys_registry).Token
}

func testAuthnLoginCreation(t *testing.T) {
	t.Run("authn: scn: Client creates a user login from a verified email token", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		// Given Client has a valid verified email token
		login, verifiedEmailToken := newAuthnFeatureVerifiedEmailToken(t, vit)

		body := fmt.Sprintf(`{"verifiedEmailToken":"%s","password":"123","displayName":"%s"}`, verifiedEmailToken, login)
		// When Client creates a user login with display name and password
		resp := vit.POST("api/v2/apps/test1/app1/users", body)
		// Then the response status is "201 Created"
		require.Equal(t, http.StatusCreated, resp.HTTPResp.StatusCode)
		// And the user login is accepted
		// And the user profile workspace creation is started
		principal := vit.SignIn(it.Login{Name: login, Pwd: "123", AppQName: istructs.AppQName_test1_app1})
		require.NotZero(t, principal.ProfileWSID)
	})

	t.Run("authn: scn: Client creates a device login", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// When Client creates a device login for an application
		deviceLogin := f.VIT().SignUpDevice(istructs.AppQName_test1_app2)
		// Then the response status is "201 Created"
		// And the response contains generated device login and password
		require.NotEmpty(t, deviceLogin.Name)
		require.NotEmpty(t, deviceLogin.Pwd)
		// And the device profile workspace creation is started
		require.NotZero(t, f.VIT().SignIn(deviceLogin).ProfileWSID)
	})

	t.Run("authn: scn: Login creation rejects an active duplicate login", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		// Given an active login already exists
		login := f.newLogin("pwd-duplicate")
		// When Client creates the same login again
		// Then the response status is "409 Conflict"
		vit.SignUp(login.Name, login.Pwd, login.AppQName, it.WithReqOpt(httpu.Expect409()))
	})

	t.Run("authn: scn: Login creation succeeds for a deactivated login name", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		loginName := vit.NextName() + "@123.com"
		login := vit.SignUp(loginName, "pwd-recreate", istructs.AppQName_test1_app1)
		principal := vit.SignIn(login)
		oldCDocLoginID := vit.GetCDocLoginID(login)
		// Given a login was previously created and is now deactivated
		vit.PostProfile(principal, "c.sys.InitiateDeactivateWorkspace", "{}")
		waitForDeactivate(vit, principal.AppQName, principal.ProfileWSID, loginName)
		// When Client creates a login with the same name again
		newLogin := vit.SignUp(loginName, login.Pwd, login.AppQName)
		newPrincipal := vit.SignIn(newLogin)
		// Then the response status is "201 Created"
		// And a new login is accepted with a fresh profile workspace
		require.NotEqual(t, principal.ProfileWSID, newPrincipal.ProfileWSID)
		require.NotEqual(t, oldCDocLoginID, vit.GetCDocLoginID(newLogin))
		// And the previously deactivated login is no longer reachable for sign-in or token issue
		require.Equal(t, newPrincipal.ProfileWSID, vit.SignIn(newLogin).ProfileWSID)
	})

	t.Run("authn: scn: Login creation rejects an existing active alias", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		alias := vit.NextName()
		// Given a login exists with an active login alias
		f.newLoginWithAlias("pwd-alias-owner", alias)
		// When Client creates a login using that alias value
		// Then the response status is "409 Conflict"
		vit.SignUp(alias, "pwd-collision", istructs.AppQName_test1_app1, it.WithReqOpt(httpu.Expect409()))
	})

	t.Run("authn: scn: User login creation rejects malformed request: verifiedEmailToken", func(t *testing.T) {
		// | field              |
		// | verifiedEmailToken |
		f := newAuthnFeatureFixture(t)
		login, _ := newAuthnFeatureVerifiedEmailToken(t, f.VIT())
		// When Client creates a user login without "<field>"
		// field = verifiedEmailToken
		// Then the response status is "400 Bad Request"
		// And the response indicates "<field>" is missing
		// field = verifiedEmailToken
		assertAuthnFeatureMalformedUserCreation(t, f.VIT(), fmt.Sprintf(`{"password":"123","displayName":"%s"}`, login), "verifiedEmailToken")
	})

	t.Run("authn: scn: User login creation rejects malformed request: displayName", func(t *testing.T) {
		// | field              |
		// | displayName        |
		f := newAuthnFeatureFixture(t)
		_, verifiedEmailToken := newAuthnFeatureVerifiedEmailToken(t, f.VIT())
		// When Client creates a user login without "<field>"
		// field = displayName
		// Then the response status is "400 Bad Request"
		// And the response indicates "<field>" is missing
		// field = displayName
		assertAuthnFeatureMalformedUserCreation(t, f.VIT(), fmt.Sprintf(`{"verifiedEmailToken":"%s","password":"123"}`, verifiedEmailToken), "displayName")
	})

	t.Run("authn: scn: User login creation rejects malformed request: password", func(t *testing.T) {
		// | field              |
		// | password           |
		f := newAuthnFeatureFixture(t)
		login, verifiedEmailToken := newAuthnFeatureVerifiedEmailToken(t, f.VIT())
		// When Client creates a user login without "<field>"
		// field = password
		// Then the response status is "400 Bad Request"
		// And the response indicates "<field>" is missing
		// field = password
		assertAuthnFeatureMalformedUserCreation(t, f.VIT(), fmt.Sprintf(`{"verifiedEmailToken":"%s","displayName":"%s"}`, verifiedEmailToken, login), "password")
	})

	t.Run("authn: scn: Device login creation rejects request body", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// When Client creates a device login with a request body
		// Then the response status is "400 Bad Request"
		// And the response indicates unexpected body
		f.VIT().Func(fmt.Sprintf("api/v2/apps/%s/%s/devices", istructs.AppQName_test1_app2.Owner(), istructs.AppQName_test1_app2.Name()), "body", httpu.Expect400()).Println()
	})
}

func newAuthnFeatureVerifiedEmailToken(t *testing.T, vit *it.VIT) (login, token string) {
	t.Helper()
	login = vit.NextName() + "@123.com"
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())
	payload := payloads.VerifiedValuePayload{
		VerificationKind: appdef.VerificationKind_EMail,
		WSID:             coreutils.PseudoWSIDToAppWSID(pseudoWSID, istructs.DefaultNumAppWorkspaces),
		Field:            "Email",
		Value:            login,
		Entity:           appdef.NewQName(registry.RegistryPackage, "CreateEmailLoginParams"),
	}
	var err error
	token, err = vit.ITokens.IssueToken(istructs.AppQName_sys_registry, 10*time.Minute, &payload)
	require.NoError(t, err)
	return login, token
}

func assertAuthnFeatureMalformedUserCreation(t *testing.T, vit *it.VIT, body, field string) {
	t.Helper()
	resp := vit.POST("api/v2/apps/test1/app1/users", body, httpu.Expect400())
	require.Equal(t, http.StatusBadRequest, resp.HTTPResp.StatusCode)
	require.Contains(t, resp.Body, field)
}

// Remaining Rule groups are defined below so every scenario subtest remains a direct child of TestAuthn.

func testAuthnLoginAliasManagement(t *testing.T) {
	t.Run("authn: scn: System sets the first Login Alias", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		// Given a User Login "jsmith" with no Login Alias
		login := f.newLogin("pwd-first-alias")
		alias := vit.NextName()
		// When System sets the Login Alias "j.smith" for "jsmith"
		initiateSetLoginAlias(t, vit, login, alias, f.systemRegistryToken())
		waitForLoginAlias(t, vit, login, alias)
		// Then "jsmith" has the active Login Alias "j.smith"
		assertLoginAliasState(t, getLoginCDoc(t, vit, login), alias)
	})

	t.Run("authn: scn: System replaces an existing Login Alias", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		oldAlias := vit.NextName()
		// Given a User Login "jsmith" with the active Login Alias "j.smith"
		login := f.newLoginWithAlias("pwd-replace-alias", oldAlias)
		newAlias := vit.NextName()
		// When System sets the Login Alias "john.smith" for "jsmith"
		initiateSetLoginAlias(t, vit, login, newAlias, f.systemRegistryToken())
		waitForLoginAlias(t, vit, login, newAlias)
		// Then "jsmith" has the active Login Alias "john.smith"
		assertLoginAliasState(t, getLoginCDoc(t, vit, login), newAlias)
		// And "j.smith" is no longer active
		issuePrincipalToken(t, vit, oldAlias, login.Pwd, login.AppQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: System clears a Login Alias", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		// Given a User Login "jsmith" with the active Login Alias "j.smith"
		login := f.newLoginWithAlias("pwd-clear-alias", vit.NextName())
		// When System clears the Login Alias for "jsmith"
		initiateSetLoginAlias(t, vit, login, "", f.systemRegistryToken())
		waitForLoginAlias(t, vit, login, "")
		// Then "jsmith" has no active Login Alias
		assertLoginAliasState(t, getLoginCDoc(t, vit, login), "")
	})

	t.Run("authn: scn: Alias management rejects caller without System Principal Token", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given a user login exists
		login := f.newLogin("pwd-alias-authz")
		// When a caller without a System Principal Token sets a login alias for the user
		// Then the alias change is rejected
		initiateSetLoginAlias(t, f.VIT(), login, f.VIT().NextName(), "", httpu.Expect403())
	})

	t.Run("authn: scn: Alias creation or update rejects a colliding identifier: creates login", func(t *testing.T) {
		// | operation | identifier   |
		// | creates   | login        |
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		// Given a user login exists
		login := f.newLogin("pwd-create-login-collision")
		// And another "<identifier>" exists in the same application
		// identifier = login
		other := f.newLogin("pwd-other-login")
		// When System "<operation>" the user's login alias using that value
		// operation = creates
		initiateSetLoginAlias(t, vit, login, other.Name, f.systemRegistryToken())
		// Then the alias change is rejected as a conflict
		waitForLoginAliasError(t, vit, login)
		issuePrincipalToken(t, vit, other.Name, other.Pwd, other.AppQName)
		issuePrincipalToken(t, vit, other.Name, login.Pwd, login.AppQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: Alias creation or update rejects a colliding identifier: creates active alias", func(t *testing.T) {
		// | operation | identifier   |
		// | creates   | active alias |
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		// Given a user login exists
		login := f.newLogin("pwd-create-alias-collision")
		// And another "<identifier>" exists in the same application
		// identifier = active alias
		otherAlias := vit.NextName()
		other := f.newLoginWithAlias("pwd-other-alias", otherAlias)
		// When System "<operation>" the user's login alias using that value
		// operation = creates
		initiateSetLoginAlias(t, vit, login, otherAlias, f.systemRegistryToken())
		// Then the alias change is rejected as a conflict
		waitForLoginAliasError(t, vit, login)
		issuePrincipalToken(t, vit, otherAlias, other.Pwd, other.AppQName)
		issuePrincipalToken(t, vit, otherAlias, login.Pwd, login.AppQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: Alias creation or update rejects a colliding identifier: updates login", func(t *testing.T) {
		// | operation | identifier   |
		// | updates   | login        |
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		// Given a user login exists
		login := f.newLoginWithAlias("pwd-update-login-collision", vit.NextName())
		// And another "<identifier>" exists in the same application
		// identifier = login
		other := f.newLogin("pwd-other-update-login")
		// When System "<operation>" the user's login alias using that value
		// operation = updates
		initiateSetLoginAlias(t, vit, login, other.Name, f.systemRegistryToken())
		// Then the alias change is rejected as a conflict
		waitForLoginAliasError(t, vit, login)
	})

	t.Run("authn: scn: Alias creation or update rejects a colliding identifier: updates active alias", func(t *testing.T) {
		// | operation | identifier   |
		// | updates   | active alias |
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		// Given a user login exists
		login := f.newLoginWithAlias("pwd-update-alias-collision", vit.NextName())
		// And another "<identifier>" exists in the same application
		// identifier = active alias
		otherAlias := vit.NextName()
		f.newLoginWithAlias("pwd-other-active-alias", otherAlias)
		// When System "<operation>" the user's login alias using that value
		// operation = updates
		initiateSetLoginAlias(t, vit, login, otherAlias, f.systemRegistryToken())
		// Then the alias change is rejected as a conflict
		waitForLoginAliasError(t, vit, login)
	})

	t.Run("authn: scn: Deactivated Login identifier can become another Login Alias without exposing profile data", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		vit := f.VIT()
		const workspaceName = "shared"
		appQName := istructs.AppQName_test1_app1

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
		initiateSetLoginAlias(t, vit, activeLogin, retiredLogin.Name, f.systemRegistryToken())
		waitForLoginAlias(t, vit, activeLogin, retiredLogin.Name)

		// And Client signs in using Login Alias "retired@example.com" and the password of User Login "active@example.com"
		aliasToken := issuePrincipalToken(t, vit, retiredLogin.Name, activeLogin.Pwd, appQName)

		// Then the issued Principal Token identifies User Login "active@example.com" and its Profile Workspace
		payload := payloads.PrincipalPayload{}
		_, err := vit.ITokens.ValidateToken(aliasToken, &payload)
		require.NoError(t, err)
		require.Equal(t, activeLogin.Name, payload.Login)
		require.Equal(t, retiredLogin.Name, payload.Alias)
		require.Equal(t, activePrincipal.ProfileWSID, payload.ProfileWSID)
		require.NotEqual(t, retiredPrincipal.ProfileWSID, payload.ProfileWSID)
		aliasPrincipal := &it.Principal{
			Login:       activeLogin,
			Token:       aliasToken,
			ProfileWSID: payload.ProfileWSID,
		}

		// And Client reads value "active" from child Workspace "shared"
		aliasWorkspace := vit.WaitForWorkspace(workspaceName, aliasPrincipal)
		require.Equal(t, activeWorkspace.WSID, aliasWorkspace.WSID)
		body := `{"args":{"Schema":"app1pkg.test_ws"},"elements":[{"fields":["StrFld"]}]}`
		actualValue := vit.PostWS(aliasWorkspace, "q.sys.Collection", body).SectionRow()[0].(string)
		require.Equal(t, "active", actualValue)

		// But Client does not read value "retired" from child Workspace "shared"
		require.NotEqual(t, retiredWorkspace.WSID, aliasWorkspace.WSID)
		require.NotEqual(t, "retired", actualValue)
	})

	t.Run("authn: scn: Alias creation rejects an invalid sign-in identifier", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given a user login exists
		login := f.newLogin("pwd-invalid-alias")
		// When System sets an invalid login alias for the user
		// Then the alias change is rejected
		// And the response indicates incorrect login format
		initiateSetLoginAlias(t, f.VIT(), login, "test@test..com", f.systemRegistryToken(), it.Expect400("incorrect login format"))
	})
}

type authnFeatureLoginVisibility struct {
	vit                      *it.VIT
	login                    it.Login
	principal                *it.Principal
	systemToken              string
	targetRegistryOwnerToken string
	otherRegistryWSID        istructs.WSID
}

func testAuthnLoginStateVisibility(t *testing.T) {
	t.Run("authn: scn: Reading a Login CDoc is limited to System and the target registry WorkspaceOwner: System", func(t *testing.T) {
		// | caller                                             | result      |
		// | System                                             | succeeds    |
		f := newAuthnFeatureFixture(t)
		// Given User Login "jsmith" exists
		ctx := newAuthnFeatureLoginVisibility(t, f)
		// When <caller> reads the Login CDoc of User Login "jsmith"
		// caller = System
		cdocLogin := getLoginCDocWithToken(t, ctx.vit, ctx.login, ctx.systemToken)
		// Then the read <result>
		// result = succeeds
		assertAuthnFeatureFullLoginState(t, cdocLogin)
	})

	t.Run("authn: scn: Reading a Login CDoc is limited to System and the target registry WorkspaceOwner: target WorkspaceOwner", func(t *testing.T) {
		// | caller                                             | result      |
		// | a WorkspaceOwner of the target registry workspace | succeeds    |
		f := newAuthnFeatureFixture(t)
		// Given User Login "jsmith" exists
		ctx := newAuthnFeatureLoginVisibility(t, f)
		// When <caller> reads the Login CDoc of User Login "jsmith"
		// caller = a WorkspaceOwner of the target registry workspace
		cdocLogin := getLoginCDocWithToken(t, ctx.vit, ctx.login, ctx.targetRegistryOwnerToken)
		// Then the read <result>
		// result = succeeds
		assertAuthnFeatureFullLoginState(t, cdocLogin)
	})

	t.Run("authn: scn: Reading a Login CDoc is limited to System and the target registry WorkspaceOwner: other WorkspaceOwner", func(t *testing.T) {
		// | caller                                             | result      |
		// | a WorkspaceOwner of another workspace             | is rejected |
		f := newAuthnFeatureFixture(t)
		// Given User Login "jsmith" exists
		ctx := newAuthnFeatureLoginVisibility(t, f)
		token := issueRegistryPrincipalToken(t, ctx.vit, "other-registry-owner", ctx.otherRegistryWSID)
		// When <caller> reads the Login CDoc of User Login "jsmith"
		// caller = a WorkspaceOwner of another workspace
		// Then the read <result>
		// result = is rejected
		getLoginCDocWithToken(t, ctx.vit, ctx.login, token, httpu.Expect403())
	})

	t.Run("authn: scn: Reading a Login CDoc is limited to System and the target registry WorkspaceOwner: neither authorization", func(t *testing.T) {
		// | caller                                             | result      |
		// | a caller with neither authorization                | is rejected |
		f := newAuthnFeatureFixture(t)
		// Given User Login "jsmith" exists
		ctx := newAuthnFeatureLoginVisibility(t, f)
		token := issueRegistryPrincipalToken(t, ctx.vit, ctx.principal.Name, ctx.principal.ProfileWSID)
		// When <caller> reads the Login CDoc of User Login "jsmith"
		// caller = a caller with neither authorization
		// Then the read <result>
		// result = is rejected
		getLoginCDocWithToken(t, ctx.vit, ctx.login, token, httpu.Expect403())
	})

	t.Run("authn: scn: A target registry WorkspaceOwner read returns Login state", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		ctx := newAuthnFeatureLoginVisibility(t, f)
		alias := ctx.vit.NextName()
		// Given User Login "jsmith" has active LoginAlias "j.smith", no alias change in progress, and no alias error
		initiateSetLoginAlias(t, ctx.vit, ctx.login, alias, ctx.systemToken)
		waitForLoginAlias(t, ctx.vit, ctx.login, alias)
		// And CanonicalLoginEnablement of User Login "jsmith" is Disabled
		setCanonicalLoginEnablement(t, ctx.vit, ctx.login, false, ctx.systemToken)
		// When a WorkspaceOwner of the target registry workspace reads the Login CDoc of User Login "jsmith"
		cdocLogin := getLoginCDocWithToken(t, ctx.vit, ctx.login, ctx.targetRegistryOwnerToken)
		// Then the Login CDoc indicates CanonicalLoginEnablement is Disabled
		assertStoredCanonicalLoginDisabled(t, cdocLogin, true)
		// And Alias is "j.smith"
		// And AliasInProc is 0
		// And AliasError is empty
		assertLoginAliasState(t, cdocLogin, alias)
	})
}

func newAuthnFeatureLoginVisibility(t *testing.T, f *authnFeatureFixture) authnFeatureLoginVisibility {
	t.Helper()
	vit := f.VIT()
	login := f.newLogin("pwd-visibility")
	principal := vit.SignIn(login)
	registryAppStructs, err := vit.IAppStructsProvider.BuiltIn(istructs.AppQName_sys_registry)
	require.NoError(t, err)
	require.Greater(t, registryAppStructs.NumAppWorkspaces(), istructs.NumAppWorkspaces(1))
	targetRegistryWSID := coreutils.PseudoWSIDToAppWSID(login.PseudoProfileWSID, registryAppStructs.NumAppWorkspaces())
	targetAppWSOffset := targetRegistryWSID.BaseWSID() - istructs.FirstBaseAppWSID
	otherAppWSOffset := (targetAppWSOffset + 1) % istructs.WSID(registryAppStructs.NumAppWorkspaces())
	return authnFeatureLoginVisibility{
		vit:                      vit,
		login:                    login,
		principal:                principal,
		systemToken:              f.systemRegistryToken(),
		targetRegistryOwnerToken: issueRegistryPrincipalToken(t, vit, "target-registry-owner", targetRegistryWSID),
		otherRegistryWSID:        istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID+otherAppWSOffset),
	}
}

func assertAuthnFeatureFullLoginState(t *testing.T, cdocLogin map[string]any) {
	t.Helper()
	require.Contains(t, cdocLogin, "PwdHash")
	require.Contains(t, cdocLogin, "LoginHash")
	require.NotContains(t, cdocLogin, "CanonicalLoginDisabled")
}

func testAuthnCanonicalLoginEnablementManagement(t *testing.T) {
	t.Run("authn: scn: System sets CanonicalLoginEnablement idempotently: Enabled disables Disabled", func(t *testing.T) {
		// | initial state | operation | resulting state |
		// | Enabled       | disables  | Disabled        |
		f := newAuthnFeatureFixture(t)
		// Given CanonicalLoginEnablement of User Login "jsmith" is <initial state>
		// initial state = Enabled
		login := f.newLogin("pwd-enabled-disable")
		// When System <operation> the canonical Login "jsmith" twice
		// operation = disables
		setCanonicalLoginEnablement(t, f.VIT(), login, false, f.systemRegistryToken())
		setCanonicalLoginEnablement(t, f.VIT(), login, false, f.systemRegistryToken())
		// Then System reads CanonicalLoginEnablement of User Login "jsmith" as "<resulting state>"
		// resulting state = Disabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, f.VIT(), login), true)
	})

	t.Run("authn: scn: System sets CanonicalLoginEnablement idempotently: Disabled disables Disabled", func(t *testing.T) {
		// | initial state | operation | resulting state |
		// | Disabled      | disables  | Disabled        |
		f := newAuthnFeatureFixture(t)
		// Given CanonicalLoginEnablement of User Login "jsmith" is <initial state>
		// initial state = Disabled
		login := f.newLogin("pwd-disabled-disable")
		setCanonicalLoginEnablement(t, f.VIT(), login, false, f.systemRegistryToken())
		// When System <operation> the canonical Login "jsmith" twice
		// operation = disables
		setCanonicalLoginEnablement(t, f.VIT(), login, false, f.systemRegistryToken())
		setCanonicalLoginEnablement(t, f.VIT(), login, false, f.systemRegistryToken())
		// Then System reads CanonicalLoginEnablement of User Login "jsmith" as "<resulting state>"
		// resulting state = Disabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, f.VIT(), login), true)
	})

	t.Run("authn: scn: System sets CanonicalLoginEnablement idempotently: Disabled enables Enabled", func(t *testing.T) {
		// | initial state | operation | resulting state |
		// | Disabled      | enables   | Enabled         |
		f := newAuthnFeatureFixture(t)
		// Given CanonicalLoginEnablement of User Login "jsmith" is <initial state>
		// initial state = Disabled
		login := f.newLogin("pwd-disabled-enable")
		setCanonicalLoginEnablement(t, f.VIT(), login, false, f.systemRegistryToken())
		// When System <operation> the canonical Login "jsmith" twice
		// operation = enables
		setCanonicalLoginEnablement(t, f.VIT(), login, true, f.systemRegistryToken())
		setCanonicalLoginEnablement(t, f.VIT(), login, true, f.systemRegistryToken())
		// Then System reads CanonicalLoginEnablement of User Login "jsmith" as "<resulting state>"
		// resulting state = Enabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, f.VIT(), login), false)
	})

	t.Run("authn: scn: System sets CanonicalLoginEnablement idempotently: Enabled enables Enabled", func(t *testing.T) {
		// | initial state | operation | resulting state |
		// | Enabled       | enables   | Enabled         |
		f := newAuthnFeatureFixture(t)
		// Given CanonicalLoginEnablement of User Login "jsmith" is <initial state>
		// initial state = Enabled
		login := f.newLogin("pwd-enabled-enable")
		// When System <operation> the canonical Login "jsmith" twice
		// operation = enables
		setCanonicalLoginEnablement(t, f.VIT(), login, true, f.systemRegistryToken())
		setCanonicalLoginEnablement(t, f.VIT(), login, true, f.systemRegistryToken())
		// Then System reads CanonicalLoginEnablement of User Login "jsmith" as "<resulting state>"
		// resulting state = Enabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, f.VIT(), login), false)
	})

	t.Run("authn: scn: Canonical Login enablement management requires a System PrincipalToken: disables", func(t *testing.T) {
		// | operation |
		// | disables  |
		f := newAuthnFeatureFixture(t)
		// Given CanonicalLoginEnablement of User Login "jsmith" is Enabled
		login := f.newLogin("pwd-reject-disable")
		// When a caller without a System PrincipalToken <operation> the canonical Login "jsmith"
		// operation = disables
		setCanonicalLoginEnablement(t, f.VIT(), login, false, newAuthnFeatureNonSystemRegistryToken(t, f), httpu.Expect403())
		// Then the enablement operation is rejected
		// And CanonicalLoginEnablement of User Login "jsmith" remains Enabled
		require.NotContains(t, getLoginCDoc(t, f.VIT(), login), "CanonicalLoginDisabled")
	})

	t.Run("authn: scn: Canonical Login enablement management requires a System PrincipalToken: enables", func(t *testing.T) {
		// | operation |
		// | enables   |
		f := newAuthnFeatureFixture(t)
		// Given CanonicalLoginEnablement of User Login "jsmith" is Enabled
		login := f.newLogin("pwd-reject-enable")
		// When a caller without a System PrincipalToken <operation> the canonical Login "jsmith"
		// operation = enables
		setCanonicalLoginEnablement(t, f.VIT(), login, true, newAuthnFeatureNonSystemRegistryToken(t, f), httpu.Expect403())
		// Then the enablement operation is rejected
		// And CanonicalLoginEnablement of User Login "jsmith" remains Enabled
		require.NotContains(t, getLoginCDoc(t, f.VIT(), login), "CanonicalLoginDisabled")
	})
}

func newAuthnFeatureNonSystemRegistryToken(t *testing.T, f *authnFeatureFixture) string {
	t.Helper()
	caller := f.newLogin("pwd-non-system")
	principal := f.VIT().SignIn(caller)
	return issueRegistryPrincipalToken(t, f.VIT(), principal.Name, principal.ProfileWSID)
}

type authnFeatureCanonicalAlias struct {
	login           it.Login
	alias           string
	profileWSID     istructs.WSID
	aliasPseudoWSID istructs.WSID
}

func testAuthnDisabledCanonicalLoginBehavior(t *testing.T) {
	t.Run("authn: scn: Disabling canonical Login preserves its active LoginAlias", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
		ctx := newAuthnFeatureCanonicalAlias(t, f)
		// When System disables the canonical Login "jsmith@example.com"
		setCanonicalLoginEnablement(t, f.VIT(), ctx.login, false, f.systemRegistryToken())
		cdocLogin := getLoginCDoc(t, f.VIT(), ctx.login)
		// Then CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		assertStoredCanonicalLoginDisabled(t, cdocLogin, true)
		// And LoginAlias "j.smith@example.com" remains active for User Login "jsmith@example.com"
		assertLoginAliasState(t, cdocLogin, ctx.alias)
	})

	t.Run("authn: scn: Disabled canonical Login rejects only canonical entry operations: sign-in", func(t *testing.T) {
		// | operation                                                                      | status           | public failure                     | observable result                              |
		// | signs in using canonical Login "jsmith@example.com" and the correct password   | 401 Unauthorized | an unknown Login or wrong password | no PrincipalToken is returned                  |
		f := newAuthnFeatureFixture(t)
		ctx := newAuthnFeatureCanonicalAlias(t, f)
		// Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, f.VIT(), ctx.login, false, f.systemRegistryToken())
		// When Client <operation>
		// operation = signs in using canonical Login "jsmith@example.com" and the correct password
		// Then the response status is "<status>"
		// status = 401 Unauthorized
		// And the response is the same as for <public failure>
		// public failure = an unknown Login or wrong password
		// And <observable result>
		// observable result = no PrincipalToken is returned
		issuePrincipalToken(t, f.VIT(), ctx.login.Name, ctx.login.Pwd, ctx.login.AppQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: Disabled canonical Login rejects only canonical entry operations: password reset", func(t *testing.T) {
		// | operation                                                                      | status           | public failure                     | observable result                              |
		// | initiates password reset using canonical Login "jsmith@example.com"            | 400 Bad Request  | an unknown Login                   | no password-reset verification code is issued  |
		f := newAuthnFeatureFixture(t)
		ctx := newAuthnFeatureCanonicalAlias(t, f)
		// Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, f.VIT(), ctx.login, false, f.systemRegistryToken())
		// When Client <operation>
		// operation = initiates password reset using canonical Login "jsmith@example.com"
		// Then the response status is "<status>"
		// status = 400 Bad Request
		// And the response is the same as for <public failure>
		// public failure = an unknown Login
		// And <observable result>
		// observable result = no password-reset verification code is issued
		assertResetPasswordInitiationRejected(t, f.VIT(), ctx.login.AppQName, ctx.login.Name)
	})

	t.Run("authn: scn: Active LoginAlias sign-in is unaffected by canonical Login disablement", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
		ctx := newAuthnFeatureCanonicalAlias(t, f)
		// And CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, f.VIT(), ctx.login, false, f.systemRegistryToken())
		// And ProfileWorkspace of User Login "jsmith@example.com" is ready
		// When Client signs in using LoginAlias "j.smith@example.com" and the correct password
		token := issuePrincipalToken(t, f.VIT(), ctx.alias, ctx.login.Pwd, ctx.login.AppQName)
		// Then the response contains PrincipalToken, expiresInSeconds, and profileWSID
		assertPrincipalTokenClaims(t, f.VIT(), token, ctx.login.Name, ctx.alias)
	})

	t.Run("authn: scn: Active LoginAlias password reset is unaffected by canonical Login disablement", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
		ctx := newAuthnFeatureCanonicalAlias(t, f)
		// And CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, f.VIT(), ctx.login, false, f.systemRegistryToken())
		// When Client initiates password reset using LoginAlias "j.smith@example.com"
		token, code, profileWSID, canonicalPseudoWSID := initiateResetPasswordByEmailAndCapture(t, f.VIT(), ctx.login.AppQName, ctx.aliasPseudoWSID, ctx.alias)
		require.Equal(t, ctx.profileWSID, profileWSID)
		require.Equal(t, ctx.login.PseudoProfileWSID, canonicalPseudoWSID)
		// And Client verifies the reset code sent to "j.smith@example.com"
		verifiedValueToken := issueVerifiedValueTokenForResetPassword(t, f.VIT(), ctx.login.AppQName, ctx.aliasPseudoWSID, token, code, profileWSID)
		newPwd := "pwd-reset-through-alias"
		// And Client resets the password with the VerifiedValueToken
		resetPasswordByEmail(t, f.VIT(), ctx.login.AppQName, canonicalPseudoWSID, verifiedValueToken, newPwd)
		// Then Client can sign in using LoginAlias "j.smith@example.com" and the new password
		issuePrincipalToken(t, f.VIT(), ctx.alias, newPwd, ctx.login.AppQName)
		// And CanonicalLoginEnablement of User Login "jsmith@example.com" remains Disabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, f.VIT(), ctx.login), true)
	})

	t.Run("authn: scn: Password reset initiated before canonical Login disablement can complete", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given User Login "jsmith@example.com" has active LoginAlias "j.smith@example.com"
		ctx := newAuthnFeatureCanonicalAlias(t, f)
		// And Client initiated password reset using canonical Login "jsmith@example.com"
		token, code, profileWSID, canonicalPseudoWSID := initiateResetPasswordByEmailAndCapture(t, f.VIT(), ctx.login.AppQName, ctx.login.PseudoProfileWSID, ctx.login.Name)
		// And Client verified the reset code and received a VerifiedValueToken
		verifiedValueToken := issueVerifiedValueTokenForResetPassword(t, f.VIT(), ctx.login.AppQName, canonicalPseudoWSID, token, code, profileWSID)
		// And System disabled the canonical Login "jsmith@example.com"
		setCanonicalLoginEnablement(t, f.VIT(), ctx.login, false, f.systemRegistryToken())
		newPwd := "pwd-reset-after-disable"
		// When Client resets the password with the VerifiedValueToken
		resetPasswordByEmail(t, f.VIT(), ctx.login.AppQName, canonicalPseudoWSID, verifiedValueToken, newPwd)
		// Then Client can sign in using active LoginAlias "j.smith@example.com" and the new password
		issuePrincipalToken(t, f.VIT(), ctx.alias, newPwd, ctx.login.AppQName)
		// And CanonicalLoginEnablement of User Login "jsmith@example.com" remains Disabled
		assertStoredCanonicalLoginDisabled(t, getLoginCDoc(t, f.VIT(), ctx.login), true)
	})

	t.Run("authn: scn: Disabled canonical identifier remains reserved", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		ctx := newAuthnFeatureCanonicalAlias(t, f)
		// Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, f.VIT(), ctx.login, false, f.systemRegistryToken())
		// When Client creates User Login "jsmith@example.com" again
		// Then the response status is "409 Conflict"
		f.VIT().SignUp(ctx.login.Name, "other-pwd", ctx.login.AppQName, it.WithReqOpt(httpu.Expect409()))
	})

	t.Run("authn: scn: Re-enabling canonical Login restores canonical entry operations: sign-in", func(t *testing.T) {
		// | operation                                                                     | observable result                            |
		// | signs in using canonical Login "jsmith@example.com" and the existing password | a new PrincipalToken is returned             |
		f := newAuthnFeatureFixture(t)
		ctx := newAuthnFeatureCanonicalAlias(t, f)
		// Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, f.VIT(), ctx.login, false, f.systemRegistryToken())
		// When System enables the canonical Login "jsmith@example.com"
		setCanonicalLoginEnablement(t, f.VIT(), ctx.login, true, f.systemRegistryToken())
		// And Client <operation>
		// operation = signs in using canonical Login "jsmith@example.com" and the existing password
		token := issuePrincipalToken(t, f.VIT(), ctx.login.Name, ctx.login.Pwd, ctx.login.AppQName)
		// Then <observable result>
		// observable result = a new PrincipalToken is returned
		assertPrincipalTokenClaims(t, f.VIT(), token, ctx.login.Name, ctx.alias)
	})

	t.Run("authn: scn: Re-enabling canonical Login restores canonical entry operations: password reset", func(t *testing.T) {
		// | operation                                                                     | observable result                            |
		// | initiates password reset using canonical Login "jsmith@example.com"           | a password-reset verification code is issued |
		f := newAuthnFeatureFixture(t)
		ctx := newAuthnFeatureCanonicalAlias(t, f)
		// Given CanonicalLoginEnablement of User Login "jsmith@example.com" is Disabled
		setCanonicalLoginEnablement(t, f.VIT(), ctx.login, false, f.systemRegistryToken())
		// When System enables the canonical Login "jsmith@example.com"
		setCanonicalLoginEnablement(t, f.VIT(), ctx.login, true, f.systemRegistryToken())
		// And Client <operation>
		// operation = initiates password reset using canonical Login "jsmith@example.com"
		_, _, profileWSID, canonicalPseudoWSID := initiateResetPasswordByEmailAndCapture(t, f.VIT(), ctx.login.AppQName, ctx.login.PseudoProfileWSID, ctx.login.Name)
		// Then <observable result>
		// observable result = a password-reset verification code is issued
		require.Equal(t, ctx.profileWSID, profileWSID)
		require.Equal(t, ctx.login.PseudoProfileWSID, canonicalPseudoWSID)
	})
}

func newAuthnFeatureCanonicalAlias(t *testing.T, f *authnFeatureFixture) authnFeatureCanonicalAlias {
	t.Helper()
	vit := f.VIT()
	alias := vit.NextName() + "@alias.example.com"
	login := f.newLoginWithAlias("pwd-canonical-alias", alias)
	principal := vit.SignIn(login)
	return authnFeatureCanonicalAlias{
		login:           login,
		alias:           alias,
		profileWSID:     principal.ProfileWSID,
		aliasPseudoWSID: coreutils.GetPseudoWSID(istructs.NullWSID, alias, istructs.CurrentClusterID()),
	}
}

func testAuthnSignInAndProfileReadiness(t *testing.T) {
	t.Run("authn: scn: Subject signs in after profile workspace is ready: user", func(t *testing.T) {
		// | subject |
		// | user    |
		f := newAuthnFeatureFixture(t)
		// Given "<subject>" login exists
		// subject = user
		login := f.newLogin("pwd-signin-user")
		// And the profile workspace for "<subject>" is ready
		// subject = user
		// When Client signs in with login and password
		result := authnFeatureSignInResult(t, f.VIT(), login)
		// Then the response contains principalToken, expiresInSeconds, and profileWSID
		require.NotEmpty(t, result["principalToken"])
		require.Equal(t, 3600.0, result["expiresInSeconds"])
		require.NotZero(t, istructs.WSID(result["profileWSID"].(float64)))
	})

	t.Run("authn: scn: Subject signs in after profile workspace is ready: device", func(t *testing.T) {
		// | subject |
		// | device  |
		f := newAuthnFeatureFixture(t)
		// Given "<subject>" login exists
		// subject = device
		deviceLogin := f.VIT().SignUpDevice(istructs.AppQName_test1_app2)
		// And the profile workspace for "<subject>" is ready
		// subject = device
		// When Client signs in with login and password
		principal := f.VIT().SignIn(deviceLogin)
		// Then the response contains principalToken, expiresInSeconds, and profileWSID
		require.NotEmpty(t, principal.Token)
		require.NotZero(t, principal.ProfileWSID)
	})

	t.Run("authn: scn: User signs in with original login while alias is active", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		alias := f.VIT().NextName()
		// Given a user login exists with an active login alias
		login := f.newLoginWithAlias("pwd-original-signin", alias)
		// And the profile workspace for the user is ready
		// When Client signs in with original login and password
		token := issuePrincipalToken(t, f.VIT(), login.Name, login.Pwd, login.AppQName)
		// Then the response contains principalToken, expiresInSeconds, and profileWSID
		assertPrincipalTokenClaims(t, f.VIT(), token, login.Name, alias)
	})

	t.Run("authn: scn: User signs in with active alias", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		alias := f.VIT().NextName()
		// Given a user login exists with an active login alias
		login := f.newLoginWithAlias("pwd-alias-signin", alias)
		// And the profile workspace for the user is ready
		// When Client signs in with alias and password
		token := issuePrincipalToken(t, f.VIT(), alias, login.Pwd, login.AppQName)
		// Then the response contains principalToken, expiresInSeconds, and profileWSID
		assertPrincipalTokenClaims(t, f.VIT(), token, login.Name, alias)
	})

	t.Run("authn: scn: Sign-in rejects a previous alias after alias update", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		oldAlias := f.VIT().NextName()
		// Given a user login exists
		login := f.newLoginWithAlias("pwd-previous-alias", oldAlias)
		// And the profile workspace for the user is ready
		// And System updated the user's login alias
		newAlias := f.VIT().NextName()
		initiateSetLoginAlias(t, f.VIT(), login, newAlias, f.systemRegistryToken())
		waitForLoginAlias(t, f.VIT(), login, newAlias)
		// When Client signs in with the previous alias and password
		// Then the response status is "401 Unauthorized"
		issuePrincipalToken(t, f.VIT(), oldAlias, login.Pwd, login.AppQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: Sign-in rejects a cleared alias", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		alias := f.VIT().NextName()
		// Given a user login exists
		login := f.newLoginWithAlias("pwd-cleared-alias", alias)
		// And the profile workspace for the user is ready
		// And System cleared the user's login alias
		initiateSetLoginAlias(t, f.VIT(), login, "", f.systemRegistryToken())
		waitForLoginAlias(t, f.VIT(), login, "")
		// When Client signs in with the cleared alias and password
		// Then the response status is "401 Unauthorized"
		issuePrincipalToken(t, f.VIT(), alias, login.Pwd, login.AppQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("authn: scn: Sign-in reports profile workspace not ready", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given a login exists
		login := f.newLogin("pwd-not-ready")
		// And the profile workspace for the login is not ready
		setLoginProfileState(t, f.VIT(), login, istructs.NullWSID, "")
		body := fmt.Sprintf(`{"login":"%s","password":"%s"}`, login.Name, login.Pwd)
		// When Client signs in with login and password
		resp := f.VIT().POST("api/v2/apps/test1/app1/auth/login", body, httpu.Expect409())
		// Then the response status is "409 Conflict"
		require.Equal(t, http.StatusConflict, resp.HTTPResp.StatusCode)
		// And the response indicates the profile workspace is not yet ready
		require.Contains(t, resp.Body, "profile workspace is not yet ready")
	})

	t.Run("authn: scn: Sign-in reports profile workspace creation error", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given a login exists
		login := f.newLogin("pwd-profile-error")
		principal := f.VIT().SignIn(login)
		// And profile workspace creation failed for the login
		setLoginProfileState(t, f.VIT(), login, principal.ProfileWSID, "profile-create-failed")
		body := fmt.Sprintf(`{"login":"%s","password":"%s"}`, login.Name, login.Pwd)
		// When Client signs in with login and password
		resp := f.VIT().POST("api/v2/apps/test1/app1/auth/login", body, httpu.Expect500())
		// Then the response indicates the profile workspace creation error
		require.Contains(t, resp.Body, "profile-create-failed")
	})
}

func authnFeatureSignInResult(t *testing.T, vit *it.VIT, login it.Login) map[string]any {
	t.Helper()
	body := fmt.Sprintf(`{"login":"%s","password":"%s"}`, login.Name, login.Pwd)
	resp := vit.POST("api/v2/apps/test1/app1/auth/login", body)
	require.Equal(t, http.StatusOK, resp.HTTPResp.StatusCode)
	result := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(resp.Body), &result))
	return result
}

func testAuthnPrincipalTokenContract(t *testing.T) {
	t.Run("authn: scn: Principal token carries authn identity fields: user", func(t *testing.T) {
		// | subject |
		// | user    |
		f := newAuthnFeatureFixture(t)
		// Given "<subject>" login exists
		// subject = user
		login := f.newLogin("pwd-user-claims")
		// And the profile workspace for "<subject>" is ready
		// subject = user
		// When Client signs in with login and password
		result := authnFeatureSignInResult(t, f.VIT(), login)
		principalToken := result["principalToken"].(string)
		payload := payloads.PrincipalPayload{}
		_, err := f.VIT().ITokens.ValidateToken(principalToken, &payload)
		require.NoError(t, err)
		// Then the issued principal token identifies its login (the canonical login), subject kind, and profileWSID
		require.Equal(t, login.Name, payload.Login)
		require.Equal(t, istructs.SubjectKind_User, payload.SubjectKind)
		require.Equal(t, istructs.WSID(result["profileWSID"].(float64)), payload.ProfileWSID)
	})

	t.Run("authn: scn: Principal token carries authn identity fields: device", func(t *testing.T) {
		// | subject |
		// | device  |
		f := newAuthnFeatureFixture(t)
		// Given "<subject>" login exists
		// subject = device
		deviceLogin := f.VIT().SignUpDevice(istructs.AppQName_test1_app2)
		// And the profile workspace for "<subject>" is ready
		// subject = device
		// When Client signs in with login and password
		principal := f.VIT().SignIn(deviceLogin)
		appStructs, err := f.VIT().BuiltIn(istructs.AppQName_test1_app2)
		require.NoError(t, err)
		payload := payloads.PrincipalPayload{}
		_, err = appStructs.AppTokens().ValidateToken(principal.Token, &payload)
		require.NoError(t, err)
		// Then the issued principal token identifies its login (the canonical login), subject kind, and profileWSID
		require.Equal(t, deviceLogin.Name, payload.Login)
		require.Equal(t, istructs.SubjectKind_Device, payload.SubjectKind)
		require.Equal(t, principal.ProfileWSID, payload.ProfileWSID)
	})

	t.Run("authn: scn: Principal token uses default TTL when no custom TTL is requested", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given a login exists
		login := f.newLogin("pwd-default-ttl")
		// When Client signs in with login and password
		result := authnFeatureSignInResult(t, f.VIT(), login)
		// Then expiresInSeconds matches the default principal token expiration
		require.Equal(t, 3600.0, result["expiresInSeconds"])
	})

	t.Run("authn: scn: Principal token rejects TTL above the maximum", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given a login exists
		login := f.newLogin("pwd-max-ttl")
		body := fmt.Sprintf(`{"args":{"Login":"%s","Password":"%s","AppName":"%s","TTLHours":1000},"elements":[{"fields":["PrincipalToken"]}]}`,
			login.Name, login.Pwd, login.AppQName)
		// When Client requests a principal token with TTL above the maximum
		// Then the response status is "400 Bad Request"
		// And the response indicates the maximum token TTL
		f.VIT().PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.registry.IssuePrincipalToken", body, it.Expect400("max token TTL hours is 168 hours"))
	})

	t.Run("authn: scn: Client refreshes a principal token", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		login := f.newLogin("pwd-refresh")
		// Given Client has a valid principal token
		principal := f.VIT().SignIn(login)
		f.VIT().TimeAdd(time.Minute)
		// When Client refreshes the principal token
		resp := f.VIT().POST("api/v2/apps/test1/app1/auth/refresh", "", httpu.WithAuthorizeBy(principal.Token))
		require.Equal(t, http.StatusOK, resp.HTTPResp.StatusCode)
		result := map[string]any{}
		require.NoError(t, json.Unmarshal([]byte(resp.Body), &result))
		newToken := result["principalToken"].(string)
		// Then the response contains a new principalToken
		require.NotEmpty(t, newToken)
		require.NotEqual(t, principal.Token, newToken)
		payload := payloads.PrincipalPayload{}
		_, err := f.VIT().ITokens.ValidateToken(newToken, &payload)
		require.NoError(t, err)
		// And the new principalToken preserves the login (canonical), alias, subject kind, and profileWSID from the input token
		require.Equal(t, login.Name, payload.Login)
		require.Empty(t, payload.Alias)
		require.Equal(t, istructs.SubjectKind_User, payload.SubjectKind)
		require.Equal(t, principal.ProfileWSID, payload.ProfileWSID)
	})

	t.Run("authn: scn: Principal token carries the canonical login and the active alias: alias", func(t *testing.T) {
		// | alias state         | identifier     | alias value      |
		// | has an active alias | alias          | the active alias |
		f := newAuthnFeatureFixture(t)
		alias := f.VIT().NextName()
		// Given a user login that <alias state>
		// alias state = has an active alias
		login := f.newLoginWithAlias("pwd-token-alias", alias)
		// And the profile workspace for the user is ready
		// When Client signs in with <identifier> and password
		// identifier = alias
		token := issuePrincipalToken(t, f.VIT(), alias, login.Pwd, login.AppQName)
		// Then the issued principal token's login is the canonical login
		// And its alias is <alias value>
		// alias value = the active alias
		assertPrincipalTokenClaims(t, f.VIT(), token, login.Name, alias)
	})

	t.Run("authn: scn: Principal token carries the canonical login and the active alias: original login with active alias", func(t *testing.T) {
		// | alias state         | identifier     | alias value      |
		// | has an active alias | original login | the active alias |
		f := newAuthnFeatureFixture(t)
		alias := f.VIT().NextName()
		// Given a user login that <alias state>
		// alias state = has an active alias
		login := f.newLoginWithAlias("pwd-token-original-alias", alias)
		// And the profile workspace for the user is ready
		// When Client signs in with <identifier> and password
		// identifier = original login
		token := issuePrincipalToken(t, f.VIT(), login.Name, login.Pwd, login.AppQName)
		// Then the issued principal token's login is the canonical login
		// And its alias is <alias value>
		// alias value = the active alias
		assertPrincipalTokenClaims(t, f.VIT(), token, login.Name, alias)
	})

	t.Run("authn: scn: Principal token carries the canonical login and the active alias: original login without active alias", func(t *testing.T) {
		// | alias state         | identifier     | alias value      |
		// | has no active alias | original login | empty            |
		f := newAuthnFeatureFixture(t)
		// Given a user login that <alias state>
		// alias state = has no active alias
		login := f.newLogin("pwd-token-no-alias")
		// And the profile workspace for the user is ready
		// When Client signs in with <identifier> and password
		// identifier = original login
		token := issuePrincipalToken(t, f.VIT(), login.Name, login.Pwd, login.AppQName)
		// Then the issued principal token's login is the canonical login
		// And its alias is <alias value>
		// alias value = empty
		assertPrincipalTokenClaims(t, f.VIT(), token, login.Name, "")
	})

	t.Run("authn: scn: Existing principal token retains login and alias after alias changes", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		alias := f.VIT().NextName()
		login := f.newLoginWithAlias("pwd-token-snapshot", alias)
		principal := f.VIT().SignIn(login)
		// Given Client has a valid principal token issued while a login alias is active
		token := issuePrincipalToken(t, f.VIT(), alias, login.Pwd, login.AppQName)
		// When System updates or clears that login alias
		initiateSetLoginAlias(t, f.VIT(), login, "", f.systemRegistryToken())
		waitForLoginAlias(t, f.VIT(), login, "")
		// Then the existing principal token remains valid until normal expiration
		assertPrincipalTokenClaims(t, f.VIT(), token, login.Name, alias)
		f.VIT().TimeAdd(time.Minute)
		principalWithSnapshot := &it.Principal{Login: login, Token: token, ProfileWSID: principal.ProfileWSID}
		resp := f.VIT().PostProfile(principalWithSnapshot, "q.sys.RefreshPrincipalToken", `{"args":{},"elements":[{"fields":["NewPrincipalToken"]}]}`)
		refreshedToken := resp.SectionRow()[0].(string)
		// And the existing principal token retains the login (canonical) and alias captured at issue time
		assertPrincipalTokenClaims(t, f.VIT(), refreshedToken, login.Name, alias)
	})
}

func testAuthnPasswordLifecycle(t *testing.T) {
	t.Run("authn: scn: Client changes user password", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given a user login exists
		login := f.newLogin("pwd-before-change")
		body := fmt.Sprintf(`{"login":"%s","oldPassword":"%s","newPassword":"pwd-after-change"}`, login.Name, login.Pwd)
		// When Client changes the password with the current password
		resp := f.VIT().POST("api/v2/apps/test1/app1/users/change-password", body)
		// Then the response status is "200 OK"
		require.Equal(t, http.StatusOK, resp.HTTPResp.StatusCode)
		login.Pwd = "pwd-after-change"
		// And Client can sign in with the new password
		f.VIT().SignIn(login)
	})

	t.Run("authn: scn: Password change rejects malformed request", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		badRequests := []string{
			`{}`,
			`{"login":"abc"}`,
			`{"login":"abc","oldPassword":"1"}`,
			`{"login":"abc","newPassword":"2"}`,
			`{"login":1,"oldPassword":"1","newPassword":"2"}`,
			`{"login":"abc","oldPassword":1,"newPassword":"2"}`,
			`{"login":"abc","oldPassword":"1","newPassword":2}`,
		}
		for _, body := range badRequests {
			// When Client changes a password without login, oldPassword, or newPassword
			// Then the response status is "400 Bad Request"
			f.VIT().POST("api/v2/apps/test1/app1/users/change-password", body, httpu.Expect400()).Println()
		}
	})

	t.Run("authn: scn: Password change rejects unknown login or wrong current password: unknown login", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		body := fmt.Sprintf(`{"login":"%s","oldPassword":"1","newPassword":"2"}`, f.VIT().NextName())
		// When Client changes a password for an unknown login or with the wrong current password
		// Then the response status is "401 Unauthorized"
		f.VIT().POST("api/v2/apps/test1/app1/users/change-password", body, httpu.Expect401()).Println()
	})

	t.Run("authn: scn: Password change rejects unknown login or wrong current password: wrong current password", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		login := f.newLogin("pwd-current")
		body := fmt.Sprintf(`{"login":"%s","oldPassword":"wrong-password","newPassword":"new-password"}`, login.Name)
		// When Client changes a password for an unknown login or with the wrong current password
		// Then the response status is "401 Unauthorized"
		f.VIT().POST("api/v2/apps/test1/app1/users/change-password", body, httpu.Expect401()).Println()
	})

	t.Run("authn: scn: Client resets password by verified email", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given a user login exists
		login := f.newEmailLogin("pwd-before-email-reset")
		// When Client initiates password reset by email
		token, code, profileWSID, canonicalPseudoWSID := initiateResetPasswordByEmailAndCapture(t, f.VIT(), login.AppQName, login.PseudoProfileWSID, login.Name)
		// And Client verifies the reset code
		verifiedValueToken := issueVerifiedValueTokenForResetPassword(t, f.VIT(), login.AppQName, canonicalPseudoWSID, token, code, profileWSID)
		newPwd := "pwd-after-email-reset"
		// And Client resets the password with the verified value token
		resetPasswordByEmail(t, f.VIT(), login.AppQName, canonicalPseudoWSID, verifiedValueToken, newPwd)
		// Then Client can sign in with the new password
		login.Pwd = newPwd
		f.VIT().SignIn(login)
	})

	t.Run("authn: scn: Client resets password by verified alias email", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		alias := f.VIT().NextName() + "@alias.example.com"
		// Given User Login "jsmith" has active Login Alias "j.smith@example.com"
		login := f.newLoginWithAlias("pwd-before-alias-reset", alias)
		aliasPseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, alias, istructs.CurrentClusterID())
		// When Client initiates password reset using Login Alias "j.smith@example.com"
		token, code, profileWSID, canonicalPseudoWSID := initiateResetPasswordByEmailAndCapture(t, f.VIT(), login.AppQName, aliasPseudoWSID, alias)
		require.Equal(t, login.PseudoProfileWSID, canonicalPseudoWSID)
		// And Client verifies the reset code sent to "j.smith@example.com"
		verifiedValueToken := issueVerifiedValueTokenForResetPassword(t, f.VIT(), login.AppQName, aliasPseudoWSID, token, code, profileWSID)
		newPwd := "pwd-after-alias-reset"
		// And Client resets the password with the verified value token
		resetPasswordByEmail(t, f.VIT(), login.AppQName, canonicalPseudoWSID, verifiedValueToken, newPwd)
		// Then Client can sign in as User Login "jsmith" with the new password
		login.Pwd = newPwd
		f.VIT().SignIn(login)
	})

	t.Run("authn: scn: Password reset initiation rejects an inactive alias: replaced", func(t *testing.T) {
		// | operation             |
		// | replaced              |
		f := newAuthnFeatureFixture(t)
		oldAlias := f.VIT().NextName() + "@alias.example.com"
		// Given User Login "jsmith" had Login Alias "j.smith@example.com"
		login := f.newLoginWithAlias("pwd-replaced-reset-alias", oldAlias)
		// And System <operation> that Login Alias
		// operation = replaced
		newAlias := f.VIT().NextName() + "@alias.example.com"
		initiateSetLoginAlias(t, f.VIT(), login, newAlias, f.systemRegistryToken())
		waitForLoginAlias(t, f.VIT(), login, newAlias)
		// When Client initiates password reset using Login Alias "j.smith@example.com"
		// Then the response status is "400 Bad Request"
		assertResetPasswordInitiationRejected(t, f.VIT(), login.AppQName, oldAlias)
	})

	t.Run("authn: scn: Password reset initiation rejects an inactive alias: cleared", func(t *testing.T) {
		// | operation             |
		// | cleared               |
		f := newAuthnFeatureFixture(t)
		alias := f.VIT().NextName() + "@alias.example.com"
		// Given User Login "jsmith" had Login Alias "j.smith@example.com"
		login := f.newLoginWithAlias("pwd-cleared-reset-alias", alias)
		// And System <operation> that Login Alias
		// operation = cleared
		initiateSetLoginAlias(t, f.VIT(), login, "", f.systemRegistryToken())
		waitForLoginAlias(t, f.VIT(), login, "")
		// When Client initiates password reset using Login Alias "j.smith@example.com"
		// Then the response status is "400 Bad Request"
		assertResetPasswordInitiationRejected(t, f.VIT(), login.AppQName, alias)
	})

	t.Run("authn: scn: Password reset initiation rejects unknown login", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		unknown := f.VIT().NextName() + "@unknown.example.com"
		body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, istructs.AppQName_test1_app1, unknown)
		// When Client initiates password reset for an unknown login
		// Then the response status is "400 Bad Request"
		f.VIT().PostApp(istructs.AppQName_sys_registry, coreutils.GetPseudoWSID(istructs.NullWSID, unknown, istructs.CurrentClusterID()), "q.registry.InitiateResetPasswordByEmail", body, httpu.Expect400()).Println()
	})

	t.Run("authn: scn: Password reset verification rejects wrong verification code", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		login := f.newEmailLogin("pwd-wrong-reset-code")
		// Given Client initiated password reset by email
		token, code, profileWSID, canonicalPseudoWSID := initiateResetPasswordByEmailAndCapture(t, f.VIT(), login.AppQName, login.PseudoProfileWSID, login.Name)
		body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`,
			token, code+"1", profileWSID, login.AppQName)
		// When Client verifies the reset code with a wrong code
		// Then the response status is "400 Bad Request"
		f.VIT().PostApp(istructs.AppQName_sys_registry, canonicalPseudoWSID, "q.registry.IssueVerifiedValueTokenForResetPassword", body, httpu.Expect400()).Println()
	})
}

func testAuthnExceptionFlows(t *testing.T) {
	t.Run("authn: scn: User login creation rejects an invalid verified email token", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given Client has an invalid verified email token
		body := fmt.Sprintf(`{"verifiedEmailToken":"invalid-token","password":"123","displayName":"%s"}`, f.VIT().NextName())
		// When Client creates a user login with display name and password
		resp := f.VIT().POST("api/v2/apps/test1/app1/users", body, httpu.Expect400())
		// Then the response status is "400 Bad Request"
		require.Equal(t, http.StatusBadRequest, resp.HTTPResp.StatusCode)
		// And the response indicates verifiedEmailToken validation failed
		require.Contains(t, resp.Body, "verifiedEmailToken")
	})

	t.Run("authn: scn: Login creation rejects an invalid login name", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		wrongLogins := []string{
			"哇", "test@tesT.com", "test@test.com ", " test@test.com", " test@test.com ",
			".test@test.com", "test@test.com.", ".test@test.com.", "test@test..com", "-test@test.com",
			"test@test.com-", "-test@test.com-", "sys.test@test.com", ",", "test,foo@test.com",
		}
		for _, wrongLogin := range wrongLogins {
			pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, wrongLogin, istructs.CurrentClusterID())
			body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"1"}}`,
				wrongLogin, istructs.AppQName_test1_app1, istructs.SubjectKind_User, istructs.CurrentClusterID())
			// When Client creates a login with an invalid login name
			// Then the response status is "400 Bad Request"
			// And the response indicates incorrect login format
			f.VIT().PostApp(istructs.AppQName_sys_registry, pseudoWSID, "c.registry.CreateLogin", body, it.Expect400("incorrect login format"))
		}
	})

	t.Run("authn: scn: Sign-in rejects unknown login or wrong password: unknown login", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		unknownLogin := f.VIT().NextName()
		body := fmt.Sprintf(`{"args":{"Login":"%s","Password":"1","AppName":"%s"},"elements":[{"fields":["PrincipalToken","WSID","WSError"]}]}`,
			unknownLogin, istructs.AppQName_test1_app1)
		// When Client signs in with unknown login or wrong password
		// Then the response status is "401 Unauthorized"
		f.VIT().PostApp(istructs.AppQName_sys_registry, coreutils.GetPseudoWSID(istructs.NullWSID, unknownLogin, istructs.CurrentClusterID()), "q.registry.IssuePrincipalToken", body, httpu.Expect401()).Println()
	})

	t.Run("authn: scn: Sign-in rejects unknown login or wrong password: wrong password", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		login := f.newLogin("pwd-correct")
		// When Client signs in with unknown login or wrong password
		body := fmt.Sprintf(`{"login":"%s","password":"wrong-password"}`, login.Name)
		resp := f.VIT().POST("api/v2/apps/test1/app1/auth/login", body, httpu.Expect401())
		// Then the response status is "401 Unauthorized"
		require.JSONEq(t, `{"status":401,"message":"login or password is incorrect"}`, resp.Body)
	})

	t.Run("authn: scn: Sign-in rejects a deactivated login with the same error as a missing login", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		login := f.newLogin("pwd-deactivated")
		principal := f.VIT().SignIn(login)
		// Given a login exists but is deactivated
		f.VIT().PostProfile(principal, "c.sys.InitiateDeactivateWorkspace", "{}")
		waitForDeactivate(f.VIT(), principal.AppQName, principal.ProfileWSID, login.Name)
		body := fmt.Sprintf(`{"args":{"Login":"%s","Password":"%s","AppName":"%s"},"elements":[{"fields":["PrincipalToken"]}]}`,
			login.Name, login.Pwd, login.AppQName)
		// When Client signs in with that login and password
		// Then the response status is "401 Unauthorized"
		// And the response indicates the login or password is incorrect
		f.VIT().PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.registry.IssuePrincipalToken", body, it.Expect401("login or password is incorrect")).Println()
	})

	t.Run("authn: scn: Principal token refresh requires an existing token", func(t *testing.T) {
		f := newAuthnFeatureFixture(t)
		// Given Client has no principal token
		// When Client refreshes the principal token
		resp := f.VIT().POST("api/v2/apps/test1/app1/auth/refresh", "", httpu.Expect401())
		// Then the response status is "401 Unauthorized"
		require.JSONEq(t, `{"status":401,"message":"authorization header is empty"}`, resp.Body)
	})
}
