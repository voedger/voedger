/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"slices"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/authnz"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_SignUpIn(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	loginName1 := vit.NextName()
	loginName2 := vit.NextName()

	login1 := vit.SignUp(loginName1, "pwd1", istructs.AppQName_test1_app1)
	login2 := vit.SignUp(loginName2, "pwd2", istructs.AppQName_test1_app1) // now wrong to create a login in a different CLusterID because it is unknown how to init AppWorkspace there

	prn1 := vit.SignIn(login1)
	prn2 := vit.SignIn(login2)

	require.NotEqual(prn1.Token, prn2.Token)
	require.Equal(istructs.ClusterID(1), prn1.ProfileWSID.ClusterID())
	require.Equal(istructs.ClusterID(1), prn2.ProfileWSID.ClusterID())
	require.True(prn1.ProfileWSID.BaseWSID() >= istructs.FirstBaseUserWSID &&
		prn2.ProfileWSID.BaseWSID() >= istructs.FirstBaseUserWSID &&
		prn1.ProfileWSID.BaseWSID() != prn2.ProfileWSID.BaseWSID())

	// refresh principal token
	// simulate delay to make the new token be different
	vit.TimeAdd(time.Minute)
	body := `{"args":{},"elements":[{"fields":["NewPrincipalToken"]}]}`
	resp := vit.PostProfile(prn1, "q.sys.RefreshPrincipalToken", body)

	refreshedPrincipalToken := resp.SectionRow()[0].(string)
	require.NotEqual(prn1.Token, refreshedPrincipalToken)

	// not need to read CDoc<Login>. Nothing to do in AppWS at all.

	var idOfCDocUserProfile int64
	t.Run("check CDoc<sys.UserProfile> at profileWSID at target app at target cluster", func(t *testing.T) {
		body := `{"args":{"Schema":"sys.UserProfile"},"elements":[{"fields":["sys.ID", "DisplayName"]}]}`
		resp := vit.PostProfile(prn1, "q.sys.Collection", body)
		require.Equal(login1.Name, resp.SectionRow()[1])
		idOfCDocUserProfile = int64(resp.SectionRow()[0].(float64))
	})

	// checking CDoc<sys.UserProfile> creation is senceless because: in wsid 1 -> 403 foridden + workspace is not initialized, in profile wsid -> singleton violation

	t.Run("modify CDoc<sys.UserProfile> after creation", func(t *testing.T) {
		body := fmt.Sprintf(`{"cuds": [{"sys.ID": %d,"fields": {"DisplayName":"new name"}}]}`, idOfCDocUserProfile)
		vit.PostProfile(prn1, "c.sys.CUD", body) // nothing to check, just expect no errors here
	})

	t.Run("check token default TTL", func(t *testing.T) {
		var p payloads.PrincipalPayload
		gp, err := vit.ITokens.ValidateToken(prn1.Token, &p)
		require.NoError(err)
		require.Equal(authnz.DefaultPrincipalTokenExpiration, gp.Duration)
	})

	t.Run("sign up and sign in with special JSON characters in password", func(t *testing.T) {
		specialLoginName := vit.NextName()
		specialLogin := vit.SignUp(specialLoginName, `p"a\ss`, istructs.AppQName_test1_app1)
		vit.SignIn(specialLogin)
	})
}

func TestTTL(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	t.Run("default TTL", func(t *testing.T) {
		prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
		var p payloads.PrincipalPayload
		gp, err := vit.ITokens.ValidateToken(prn.Token, &p)
		require.NoError(err)
		require.Equal(authnz.DefaultPrincipalTokenExpiration, gp.Duration)
	})

	t.Run("custom TTL", func(t *testing.T) {
		prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
		body := fmt.Sprintf(`{"args": {"Login": %q,"Password": %q,"AppName": %q, "TTLHours":15},"elements":[{"fields":["PrincipalToken"]}]}`,
			prn.Name, prn.Pwd, prn.AppQName.String())
		resp := vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssuePrincipalToken", body)
		token := resp.SectionRow()[0].(string)
		var p payloads.PrincipalPayload
		gp, err := vit.ITokens.ValidateToken(token, &p)
		require.NoError(err)
		require.Equal(15*time.Hour, gp.Duration)
	})
}

func TestCreateLoginErrors(t *testing.T) {
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
		body := fmt.Sprintf(`{"args":{"Login":%q,"AppName":"my/unknown","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"password"}}`,
			login, istructs.SubjectKind_User, istructs.CurrentClusterID())
		vit.PostApp(istructs.AppQName_sys_registry, loginPseudoWSID, "c.registry.CreateLogin", body, it.Expect400("my/unknown is not found"))
	})

	t.Run("wrong application name", func(t *testing.T) {
		body := fmt.Sprintf(`{"args":{"Login":%q,"AppName":"wrong-AppName","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":1},"unloggedArgs":{"Password":"different"}}`,
			login, istructs.SubjectKind_User)
		vit.PostApp(istructs.AppQName_sys_registry, loginPseudoWSID, "c.registry.CreateLogin", body,
			it.Expect400("failed to parse app qualified name"))
	})

	newLogin := vit.SignUp(login, "1", istructs.AppQName_test1_app1)
	// wait for acomplishing the profile init
	vit.SignIn(newLogin)

	t.Run("create an existing login again", func(t *testing.T) {
		vit.SignUp(login, "1", istructs.AppQName_test1_app1, it.WithReqOpt(httpu.Expect409()))
	})

	t.Run("subject name constraint violation", func(t *testing.T) {
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
			body := fmt.Sprintf(`{"args":{"Login":%q,"AppName":%q,"SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":%q}}`,
				wrongLogin, istructs.AppQName_test1_app1.String(), istructs.SubjectKind_User, istructs.CurrentClusterID(), "1")
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

func TestSignInErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	login := vit.NextName()
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())

	t.Run("unknown login", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Login": %q,"Password": "1","AppName": %q},"elements":[{"fields":["PrincipalToken", "WSID", "WSError"]}]}`,
			login, istructs.AppQName_test1_app1.String())
		vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.registry.IssuePrincipalToken", body, httpu.Expect401()).Println()
	})

	newLogin := vit.SignUp(login, "1", istructs.AppQName_test1_app1)
	// wait for acomplishing the profile init
	vit.SignIn(newLogin)

	t.Run("wrong password", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"Login": %q,"Password": "wrongPass","AppName": %q},"elements":[{"fields":[]}]}`,
			login, istructs.AppQName_test1_app1.String())
		vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.registry.IssuePrincipalToken", body, httpu.Expect401()).Println()
	})

	t.Run("wrong TTL", func(t *testing.T) {
		prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
		body := fmt.Sprintf(`{"args": {"Login": %q,"Password": %q,"AppName": %q, "TTLHours":1000},"elements":[{"fields":["PrincipalToken"]}]}`,
			prn.Name, prn.Pwd, prn.AppQName.String())
		vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssuePrincipalToken", body,
			it.Expect400("max token TTL hours is 168 hours"))
	})
}

func TestLoginAlias(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	appQName := istructs.AppQName_test1_app1
	login := vit.SignUp(vit.NextName(), "pwd1", appQName)
	vit.SignIn(login)

	alias1 := vit.NextName()
	alias2 := vit.NextName()
	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token

	t.Run("alias management rejected without system principal token", func(t *testing.T) {
		initiateSetLoginAlias(t, vit, login, alias1, "", httpu.Expect403())
	})

	t.Run("set alias and sign in with original login and alias", func(t *testing.T) {
		initiateSetLoginAlias(t, vit, login, alias1, sysRegistryToken)
		waitForLoginAlias(t, vit, login, alias1)

		primaryToken := issuePrincipalToken(t, vit, login.Name, login.Pwd, appQName)
		assertPrincipalTokenClaims(t, vit, primaryToken, login.Name, alias1)

		aliasToken := issuePrincipalToken(t, vit, alias1, login.Pwd, appQName)
		assertPrincipalTokenClaims(t, vit, aliasToken, login.Name, alias1)

		issuePrincipalToken(t, vit, alias1, "wrong-password", appQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("setting the same alias is idempotent", func(t *testing.T) {
		initiateSetLoginAlias(t, vit, login, alias1, sysRegistryToken)
		waitForLoginAlias(t, vit, login, alias1)

		cdocLogin := getLoginCDoc(t, vit, login)
		require.Empty(cdocLogin["AliasError"])
		require.Equal(float64(0), cdocLogin["AliasInProc"])
	})

	t.Run("update alias rejects previous alias sign-in", func(t *testing.T) {
		initiateSetLoginAlias(t, vit, login, alias2, sysRegistryToken)
		waitForLoginAlias(t, vit, login, alias2)

		issuePrincipalToken(t, vit, alias1, login.Pwd, appQName, it.Expect401("login or password is incorrect"))
		aliasToken := issuePrincipalToken(t, vit, alias2, login.Pwd, appQName)
		assertPrincipalTokenClaims(t, vit, aliasToken, login.Name, alias2)
	})

	t.Run("existing token keeps alias snapshot and refresh preserves it", func(t *testing.T) {
		tokenBeforeClear := issuePrincipalToken(t, vit, alias2, login.Pwd, appQName)
		assertPrincipalTokenClaims(t, vit, tokenBeforeClear, login.Name, alias2)

		initiateSetLoginAlias(t, vit, login, "", sysRegistryToken)
		waitForLoginAlias(t, vit, login, "")
		issuePrincipalToken(t, vit, alias2, login.Pwd, appQName, it.Expect401("login or password is incorrect"))

		assertPrincipalTokenClaims(t, vit, tokenBeforeClear, login.Name, alias2)

		vit.TimeAdd(time.Minute)
		prnWithAliasSnapshot := &it.Principal{
			Login:       login,
			Token:       tokenBeforeClear,
			ProfileWSID: vit.SignIn(login).ProfileWSID,
		}
		body := `{"args":{},"elements":[{"fields":["NewPrincipalToken"]}]}`
		resp := vit.PostProfile(prnWithAliasSnapshot, "q.sys.RefreshPrincipalToken", body)
		refreshedToken := resp.SectionRow()[0].(string)
		require.NotEqual(tokenBeforeClear, refreshedToken)
		assertPrincipalTokenClaims(t, vit, refreshedToken, login.Name, alias2)
	})

	t.Run("clearing when no alias is set is idempotent", func(t *testing.T) {
		initiateSetLoginAlias(t, vit, login, "", sysRegistryToken)
		waitForLoginAlias(t, vit, login, "")

		cdocLogin := getLoginCDoc(t, vit, login)
		require.Empty(cdocLogin["AliasError"])
		require.Equal(float64(0), cdocLogin["AliasInProc"])
	})

	t.Run("cleared alias can be reused by another login", func(t *testing.T) {
		reuseLogin := vit.SignUp(vit.NextName(), "pwd-reuse", appQName)
		vit.SignIn(reuseLogin)

		initiateSetLoginAlias(t, vit, reuseLogin, alias2, sysRegistryToken)
		waitForLoginAlias(t, vit, reuseLogin, alias2)

		issuePrincipalToken(t, vit, alias2, login.Pwd, appQName, it.Expect401("login or password is incorrect"))
		aliasToken := issuePrincipalToken(t, vit, alias2, reuseLogin.Pwd, appQName)
		assertPrincipalTokenClaims(t, vit, aliasToken, reuseLogin.Name, alias2)
	})
}

func TestLoginAliasCommandEdgeCases(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	appQName := istructs.AppQName_test1_app1
	login := vit.SignUp(vit.NextName(), "pwd1", appQName)
	vit.SignIn(login)

	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token

	t.Run("unknown source login rejected", func(t *testing.T) {
		unknownLogin := vit.NextName()
		pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, unknownLogin, istructs.CurrentClusterID())
		initiateSetLoginAliasByName(t, vit, pseudoWSID, unknownLogin, appQName, vit.NextName(), sysRegistryToken, it.Expect401("does not exist"))
	})

	t.Run("wrong pseudo workspace rejected", func(t *testing.T) {
		crc16 := coreutils.CRC16([]byte(login.Name)) - 1
		wrongPseudoWSID := istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(crc16))
		initiateSetLoginAliasByName(t, vit, wrongPseudoWSID, login.Name, appQName, vit.NextName(), sysRegistryToken, it.Expect403("wrong AppWSID"))
	})

	t.Run("in-progress alias update rejected", func(t *testing.T) {
		setLoginAliasInProc(t, vit, login, 1)
		initiateSetLoginAlias(t, vit, login, vit.NextName(), sysRegistryToken, it.Expect409("already in progress"))
		setLoginAliasInProc(t, vit, login, 0)
	})

	t.Run("clearing without an existing alias completes", func(t *testing.T) {
		clearLogin := vit.SignUp(vit.NextName(), "pwd-clear", appQName)
		vit.SignIn(clearLogin)

		initiateSetLoginAlias(t, vit, clearLogin, "", sysRegistryToken)
		waitForLoginAlias(t, vit, clearLogin, "")
	})
}

func TestLoginAliasCollisionsAndValidation(t *testing.T) {
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

	t.Run("alias rejected when it violates sign-in identifier format rules", func(t *testing.T) {
		invalidAliasLogin := vit.SignUp(vit.NextName(), "pwd3", appQName)
		vit.SignIn(invalidAliasLogin)
		initiateSetLoginAlias(t, vit, invalidAliasLogin, "test@test..com", sysRegistryToken, it.Expect400("incorrect login format"))
	})

	t.Run("alias rejected when it collides with an existing login", func(t *testing.T) {
		collisionLogin := vit.SignUp(vit.NextName(), "pwd4", appQName)
		vit.SignIn(collisionLogin)

		initiateSetLoginAlias(t, vit, collisionLogin, login2.Name, sysRegistryToken)
		waitForLoginAliasError(t, vit, collisionLogin)
		issuePrincipalToken(t, vit, login2.Name, login2.Pwd, appQName)
		issuePrincipalToken(t, vit, login2.Name, collisionLogin.Pwd, appQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("alias rejected when it collides with an existing alias", func(t *testing.T) {
		collisionLogin := vit.SignUp(vit.NextName(), "pwd5", appQName)
		vit.SignIn(collisionLogin)

		initiateSetLoginAlias(t, vit, collisionLogin, alias1, sysRegistryToken)
		waitForLoginAliasError(t, vit, collisionLogin)
		issuePrincipalToken(t, vit, alias1, login1.Pwd, appQName)
		issuePrincipalToken(t, vit, alias1, collisionLogin.Pwd, appQName, it.Expect401("login or password is incorrect"))
	})

	t.Run("login creation rejected when requested login collides with active alias", func(t *testing.T) {
		vit.SignUp(alias1, "pwd6", appQName, it.WithReqOpt(httpu.Expect409()))
	})
}

// [~server.devices/it.TestDevicesCreate~impl]
func TestCreateDevice(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	deviceLogin := vit.SignUpDevice(istructs.AppQName_test1_app2)

	// APIv2 create device returns generated device login and password
	log.Println(deviceLogin.Name)
	log.Println(deviceLogin.Pwd)

	devicePrn := vit.SignIn(deviceLogin)
	as, err := vit.BuiltIn(istructs.AppQName_test1_app2)
	require.NoError(err)
	devicePrnPayload := payloads.PrincipalPayload{}
	_, err = as.AppTokens().ValidateToken(devicePrn.Token, &devicePrnPayload)
	require.NoError(err)
	require.Equal(istructs.SubjectKind_Device, devicePrnPayload.SubjectKind)

	t.Run("exec a simple operation in the device profile", func(t *testing.T) {
		body := `{"args":{"Schema":"sys.WorkspaceDescriptor"},"elements":[{"fields":["sys.ID"]}]}`
		vit.PostProfile(devicePrn, "q.sys.Collection", body)
	})

	t.Run("refresh the device principal token", func(t *testing.T) {
		// simulate delay to make the new token be different
		vit.TimeAdd(time.Minute)
		body := `{"args":{},"elements":[{"fields":["NewPrincipalToken"]}]}`
		resp := vit.PostProfile(devicePrn, "q.sys.RefreshPrincipalToken", body)
		require.NotEqual(devicePrn.Token, resp.SectionRow()[0].(string))
	})

	t.Run("400 bad request on an unexpected body", func(t *testing.T) {
		vit.Func(fmt.Sprintf("api/v2/apps/%s/%s/devices", deviceLogin.AppQName.Owner(), deviceLogin.AppQName.Name()), "body",
			httpu.Expect400()).Println()
	})
}

func initiateSetLoginAlias(t *testing.T, vit *it.VIT, login it.Login, alias, token string, opts ...httpu.ReqOptFunc) {
	t.Helper()
	initiateSetLoginAliasByName(t, vit, login.PseudoProfileWSID, login.Name, login.AppQName, alias, token, opts...)
}

func initiateSetLoginAliasByName(t *testing.T, vit *it.VIT, wsid istructs.WSID, login string, appQName appdef.AppQName, alias, token string, opts ...httpu.ReqOptFunc) {
	t.Helper()
	bodyBytes, err := json.Marshal(map[string]any{
		"args": map[string]any{
			"Login":   login,
			"AppName": appQName.String(),
			"Alias":   alias,
		},
	})
	require.NoError(t, err)
	if len(token) > 0 {
		opts = append(opts, httpu.WithAuthorizeBy(token))
	}
	vit.PostApp(istructs.AppQName_sys_registry, wsid, "c.registry.InitiateSetLoginAlias", string(bodyBytes), opts...)
}

func setLoginAliasInProc(t *testing.T, vit *it.VIT, login it.Login, aliasInProc int32) {
	t.Helper()
	cdocLoginID := vit.GetCDocLoginID(login)
	body := fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"AliasInProc":%d}}]}`, cdocLoginID, aliasInProc)
	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.sys.CUD", body, httpu.WithAuthorizeBy(sysRegistryToken))
}

func issuePrincipalToken(t *testing.T, vit *it.VIT, signInIdentifier, pwd string, appQName appdef.AppQName, opts ...httpu.ReqOptFunc) string {
	t.Helper()
	bodyBytes, err := json.Marshal(map[string]any{
		"args": map[string]any{
			"Login":    signInIdentifier,
			"Password": pwd,
			"AppName":  appQName.String(),
		},
		"elements": []map[string]any{{
			"fields": []string{"PrincipalToken", "WSID", "WSError"},
		}},
	})
	require.NoError(t, err)
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, signInIdentifier, istructs.CurrentClusterID())
	resp := vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.registry.IssuePrincipalToken", string(bodyBytes), opts...)
	if resp.SysError != nil {
		return ""
	}
	return resp.SectionRow()[0].(string)
}

func assertPrincipalTokenClaims(t *testing.T, vit *it.VIT, token, expectedLogin, expectedAlias string) {
	t.Helper()
	payload := payloads.PrincipalPayload{}
	_, err := vit.ITokens.ValidateToken(token, &payload)
	require.NoError(t, err)
	require.Equal(t, expectedLogin, payload.Login)

	claims := decodeJWTClaims(t, token)
	require.Equal(t, expectedAlias, claims["Alias"])
}

func decodeJWTClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	claims := map[string]any{}
	require.NoError(t, json.Unmarshal(claimsBytes, &claims))
	return claims
}

func waitForLoginAlias(t *testing.T, vit *it.VIT, login it.Login, expectedAlias string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		cdocLogin := getLoginCDoc(t, vit, login)
		if cdocLogin["Alias"] == expectedAlias && cdocLogin["AliasInProc"] == float64(0) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("login alias was not updated to %q", expectedAlias)
}

func waitForLoginAliasError(t *testing.T, vit *it.VIT, login it.Login) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		cdocLogin := getLoginCDoc(t, vit, login)
		if aliasError, ok := cdocLogin["AliasError"].(string); ok && len(aliasError) > 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("login alias error was not recorded")
}

func getLoginCDoc(t *testing.T, vit *it.VIT, login it.Login) map[string]any {
	t.Helper()
	cdocLoginID := vit.GetCDocLoginID(login)
	body := fmt.Sprintf(`{"args":{"Query":"select * from registry.Login.%d"},"elements":[{"fields":["Result"]}]}`, cdocLoginID)
	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	resp := vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.sys.SqlQuery", body, httpu.WithAuthorizeBy(sysRegistryToken))
	cdocLogin := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &cdocLogin))
	return cdocLogin
}

func TestWorkInForeignProfileWithEnrichedToken(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// create new login
	newLoginName := vit.NextName()
	newLogin := vit.SignUp(newLoginName, "1", istructs.AppQName_test1_app1)
	newLoginPrn := vit.SignIn(newLogin)

	existingLoginPrn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")

	// new login can not work in the profile of the existingLogin
	body := `{"args":{"Schema":"sys.UserProfile"},"elements":[{"fields":["sys.ID", "DisplayName"]}]}`
	vit.PostApp(istructs.AppQName_test1_app1, existingLoginPrn.ProfileWSID, "q.sys.Collection", body, httpu.Expect403(), httpu.WithAuthorizeBy(newLoginPrn.Token))

	// now enrich the token of the newLogin: make it ProfileOwner in the profile of the existingLogin

	// determine ownerWSID of the existingLogin
	body = `{"args":{"Schema":"sys.WorkspaceDescriptor"},"elements":[{"fields":["OwnerWSID"]}]}`
	resp := vit.PostProfile(existingLoginPrn, "q.sys.Collection", body)
	existingLoginOwnerWSID := istructs.WSID(resp.SectionRow()[0].(float64))

	// enrich the existing token of the newLogin with role.sys.ProfileOwner
	profileOwnerRole := payloads.RoleType{
		WSID:  existingLoginOwnerWSID,
		QName: iauthnz.QNameRoleProfileOwner,
	}
	enrichedToken := vit.EnrichPrincipalToken(newLoginPrn, []payloads.RoleType{profileOwnerRole})

	// no newLogin is able to work in the profile of the existingLogin role.sys.ProfileOwner principal is emitted for him there
	body = `{"args":{"Schema":"sys.UserProfile"},"elements":[{"fields":["sys.ID", "DisplayName"]}]}`
	vit.PostApp(istructs.AppQName_test1_app1, existingLoginPrn.ProfileWSID, "q.sys.Collection", body, httpu.WithAuthorizeBy(enrichedToken))
}

// [~server.authnz.groles/it.TestGlobalRoles~impl]
func TestGlobalRoles(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	loginName := vit.NextName()
	login := vit.SignUp(loginName, "pwd1", istructs.AppQName_test1_app1)
	prn := vit.SignIn(login)

	// no global roles in the old token
	as, err := vit.BuiltIn(istructs.AppQName_test1_app1)
	require.NoError(err)
	payload1 := payloads.PrincipalPayload{}
	_, err = as.AppTokens().ValidateToken(prn.Token, &payload1)
	require.NoError(err)
	require.Empty(payload1.GlobalRoles)

	// view is not available for the user without global roles
	vit.GET(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?where={"Year":2025}`, ws.WSID, it.QNameApp1_ViewDailyIdx),
		httpu.WithAuthorizeBy(prn.Token), httpu.Expect403())

	// update global roles not allowed by default
	body := fmt.Sprintf(`{"args":{"Login":%q,"AppName":%q,"GlobalRoles":"app1pkg.LimitedAccessRole,sys.role2"},"elements":[]}`, login.Name, login.AppQName.String())
	vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.registry.UpdateGlobalRoles", body, httpu.Expect403())

	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	// incorrect role name
	body = fmt.Sprintf(`{"args":{"Login":%q,"AppName":%q,"GlobalRoles":"LimitedAccessRole,sys.role2"},"elements":[]}`, login.Name, login.AppQName.String())
	vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.registry.UpdateGlobalRoles", body, httpu.WithAuthorizeBy(sysRegistryToken), httpu.Expect400())

	// update global roles allowed for the System principal
	body = fmt.Sprintf(`{"args":{"Login":%q,"AppName":%q,"GlobalRoles":"app1pkg.LimitedAccessRole,sys.role2"}}`, login.Name, login.AppQName.String())
	vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.registry.UpdateGlobalRoles", body, httpu.WithAuthorizeBy(sysRegistryToken))

	// now global roles are in the new token
	prn2 := vit.SignIn(login)
	payload2 := payloads.PrincipalPayload{}
	_, err = as.AppTokens().ValidateToken(prn2.Token, &payload2)
	require.NoError(err)
	require.Len(payload2.GlobalRoles, 2)
	require.True(slices.Contains(payload2.GlobalRoles, appdef.NewQName("app1pkg", "LimitedAccessRole")))
	require.True(slices.Contains(payload2.GlobalRoles, appdef.NewQName("sys", "role2")))

	// now user can work with the view
	vit.GET(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/views/%s?keys=Year,Month,Day&where={"Year":2025}`, ws.WSID, it.QNameApp1_ViewDailyIdx),
		httpu.WithAuthorizeBy(prn2.Token))

}
