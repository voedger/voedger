/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

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

	t.Run("custom TTL", func(t *testing.T) {
		prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
		body := fmt.Sprintf(`{"args": {"Login": "%s","Password": "%s","AppName": "%s", "TTLHours":15},"elements":[{"fields":["PrincipalToken"]}]}`,
			prn.Name, prn.Pwd, prn.AppQName.String())
		resp := vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "q.registry.IssuePrincipalToken", body)
		token := resp.SectionRow()[0].(string)
		var p payloads.PrincipalPayload
		gp, err := vit.ITokens.ValidateToken(token, &p)
		require.NoError(err)
		require.Equal(15*time.Hour, gp.Duration)
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

func setCanonicalLoginEnablement(t *testing.T, vit *it.VIT, login it.Login, enabled bool, token string, opts ...httpu.ReqOptFunc) {
	t.Helper()
	bodyBytes, err := json.Marshal(map[string]any{
		"args": map[string]any{
			"Login":   login.Name,
			"AppName": login.AppQName.String(),
			"Enabled": enabled,
		},
	})
	require.NoError(t, err)
	if len(token) > 0 {
		opts = append(opts, httpu.WithAuthorizeBy(token))
	}
	vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.registry.SetCanonicalLoginEnablement", string(bodyBytes), opts...)
}

func setLoginAliasInProc(t *testing.T, vit *it.VIT, login it.Login, aliasInProc int32) {
	t.Helper()
	cdocLoginID := vit.GetCDocLoginID(login)
	body := fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"AliasInProc":%d}}]}`, cdocLoginID, aliasInProc)
	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.sys.CUD", body, httpu.WithAuthorizeBy(sysRegistryToken))
}

func setLoginProfileState(t *testing.T, vit *it.VIT, login it.Login, profileWSID istructs.WSID, wsError string) {
	t.Helper()
	bodyBytes, err := json.Marshal(map[string]any{
		"cuds": []map[string]any{{
			"sys.ID": vit.GetCDocLoginID(login),
			"fields": map[string]any{
				authnz.Field_WSID:    profileWSID,
				authnz.Field_WSError: wsError,
			},
		}},
	})
	require.NoError(t, err)
	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.sys.CUD", string(bodyBytes), httpu.WithAuthorizeBy(sysRegistryToken))
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
	require.Equal(t, expectedAlias, payload.Alias)

	claims := decodeJWTClaims(t, token)
	require.Equal(t, expectedLogin, claims["Login"])
	if expectedAlias == "" {
		require.Empty(t, claims["Alias"])
	} else {
		require.Equal(t, expectedAlias, claims["Alias"])
	}
	_, hasCanonical := claims["CanonicalLogin"]
	require.False(t, hasCanonical)
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
		aliasMatches := cdocLogin["Alias"] == expectedAlias || expectedAlias == "" && cdocLogin["Alias"] == nil
		if aliasMatches && cdocLogin["AliasInProc"] == float64(0) {
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
	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	return getLoginCDocWithToken(t, vit, login, sysRegistryToken)
}

func getLoginCDocWithToken(t *testing.T, vit *it.VIT, login it.Login, token string, opts ...httpu.ReqOptFunc) map[string]any {
	t.Helper()
	cdocLoginID := vit.GetCDocLoginID(login)
	body := fmt.Sprintf(`{"args":{"ID":%d},"elements":[{"fields":["Result"]}]}`, cdocLoginID)
	if len(token) > 0 {
		opts = append(opts, httpu.WithAuthorizeBy(token))
	}
	resp := vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "q.sys.GetCDoc", body, opts...)
	if resp.SysError != nil {
		return nil
	}
	cdocLogin := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(resp.SectionRow()[0].(string)), &cdocLogin))
	return cdocLogin
}

func assertStoredCanonicalLoginDisabled(t *testing.T, cdocLogin map[string]any, expected bool) {
	t.Helper()
	actual, ok := cdocLogin["CanonicalLoginDisabled"].(bool)
	require.True(t, ok)
	require.Equal(t, expected, actual)
}

func assertLoginAliasState(t *testing.T, cdocLogin map[string]any, expectedAlias string) {
	t.Helper()
	if expectedAlias == "" {
		require.Empty(t, cdocLogin["Alias"])
	} else {
		require.Equal(t, expectedAlias, cdocLogin["Alias"])
	}
	require.Equal(t, float64(0), cdocLogin["AliasInProc"])
	require.Empty(t, cdocLogin["AliasError"])
}

func issueRegistryPrincipalToken(t *testing.T, vit *it.VIT, login string, profileWSID istructs.WSID) string {
	t.Helper()
	token, err := vit.ITokens.IssueToken(istructs.AppQName_sys_registry, time.Minute, &payloads.PrincipalPayload{
		Login:       login,
		SubjectKind: istructs.SubjectKind_User,
		ProfileWSID: profileWSID,
	})
	require.NoError(t, err)
	return token
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
	body := fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","GlobalRoles":"app1pkg.LimitedAccessRole,sys.role2"},"elements":[]}`, login.Name, login.AppQName.String())
	vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.registry.UpdateGlobalRoles", body, httpu.Expect403())

	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	// incorrect role name
	body = fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","GlobalRoles":"LimitedAccessRole,sys.role2"},"elements":[]}`, login.Name, login.AppQName.String())
	vit.PostApp(istructs.AppQName_sys_registry, prn.PseudoProfileWSID, "c.registry.UpdateGlobalRoles", body, httpu.WithAuthorizeBy(sysRegistryToken), httpu.Expect400())

	// update global roles allowed for the System principal
	body = fmt.Sprintf(`{"args":{"Login":"%s","AppName":"%s","GlobalRoles":"app1pkg.LimitedAccessRole,sys.role2"}}`, login.Name, login.AppQName.String())
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
