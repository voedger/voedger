/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package sys_it

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/authnz"
	it "github.com/voedger/voedger/pkg/vit"
)

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
	opts = appendAuthorization(opts, token)
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
	opts = appendAuthorization(opts, token)
	vit.PostApp(istructs.AppQName_sys_registry, login.PseudoProfileWSID, "c.registry.SetCanonicalLoginEnablement", string(bodyBytes), opts...)
}

func appendAuthorization(opts []httpu.ReqOptFunc, token string) []httpu.ReqOptFunc {
	if token == "" {
		return opts
	}
	return append(opts, httpu.WithAuthorizeBy(token))
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
	opts = appendAuthorization(opts, token)
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

func signUpLoginWithAlias(t *testing.T, vit *it.VIT, appQName appdef.AppQName, pwd, alias string) it.Login {
	t.Helper()
	login := vit.SignUp(vit.NextName()+"@123.com", pwd, appQName)
	vit.SignIn(login)

	sysRegistryToken := vit.GetSystemPrincipal(istructs.AppQName_sys_registry).Token
	initiateSetLoginAlias(t, vit, login, alias, sysRegistryToken)
	waitForLoginAlias(t, vit, login, alias)

	return login
}

func initiateResetPasswordByEmailAndCapture(t *testing.T, vit *it.VIT, appQName appdef.AppQName, wsid istructs.WSID, email string) (token, code string, profileWSID, canonicalPseudoWSID istructs.WSID) {
	t.Helper()
	body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID","CanonicalPseudoWSID"]}]}`, appQName, email)
	resp := vit.PostApp(istructs.AppQName_sys_registry, wsid, "q.registry.InitiateResetPasswordByEmail", body)

	emailMessage := vit.CaptureEmail()
	if len(emailMessage.To) != 1 || emailMessage.To[0] != email {
		t.Fatalf("reset code recipients = %v, want [%s]", emailMessage.To, email)
	}
	code = regexp.MustCompile(`\d{6}`).FindString(emailMessage.Body)
	if code == "" {
		t.Fatalf("reset code was not found in email body %q", emailMessage.Body)
	}

	row := resp.SectionRow()
	token = row[0].(string)
	profileWSID = istructs.WSID(row[1].(float64))
	canonicalPseudoWSID = istructs.WSID(row[2].(float64))
	return
}

func issueVerifiedValueTokenForResetPassword(t *testing.T, vit *it.VIT, appQName appdef.AppQName, wsid istructs.WSID, token, code string, profileWSID istructs.WSID) string {
	t.Helper()
	body := fmt.Sprintf(`{"args":{"VerificationToken":"%s","VerificationCode":"%s","ProfileWSID":%d,"AppName":"%s"},"elements":[{"fields":["VerifiedValueToken"]}]}`,
		token, code, profileWSID, appQName)
	resp := vit.PostApp(istructs.AppQName_sys_registry, wsid, "q.registry.IssueVerifiedValueTokenForResetPassword", body)
	return resp.SectionRow()[0].(string)
}

func resetPasswordByEmail(t *testing.T, vit *it.VIT, appQName appdef.AppQName, wsid istructs.WSID, verifiedValueToken, newPwd string) {
	t.Helper()
	body := fmt.Sprintf(`{"args":{"AppName":"%s"},"unloggedArgs":{"Email":"%s","NewPwd":"%s"}}`, appQName, verifiedValueToken, newPwd)
	vit.PostApp(istructs.AppQName_sys_registry, wsid, "c.registry.ResetPasswordByEmail", body)
}

func assertResetPasswordInitiationRejected(t *testing.T, vit *it.VIT, appQName appdef.AppQName, email string) {
	t.Helper()
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, email, istructs.CurrentClusterID())
	body := fmt.Sprintf(`{"args":{"AppName":"%s","Email":"%s"},"elements":[{"fields":["VerificationToken","ProfileWSID"]}]}`, appQName, email)
	vit.PostApp(istructs.AppQName_sys_registry, pseudoWSID, "q.registry.InitiateResetPasswordByEmail", body, it.Expect400("login does not exist"))
}
