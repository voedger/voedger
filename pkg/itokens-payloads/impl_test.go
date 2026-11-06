/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package payloads

import (
	"encoding/json"
	"log"
	"slices"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	"github.com/voedger/voedger/pkg/itokensjwt"
)

var (
	testApp      = istructs.AppQName_test1_app1
	testDuration = time.Minute
)

func TestBasicUsage_PrincipalPayload(t *testing.T) {

	require := require.New(t)

	signer := itokensjwt.TestTokensJWT()

	srcPayload := PrincipalPayload{
		Login:       "login",
		SubjectKind: istructs.SubjectKind_User,
		ProfileWSID: istructs.WSID(10),
	}

	var token string
	var err error

	t.Run("Prepare token", func(t *testing.T) {
		token, err = signer.IssueToken(testApp, testDuration, &srcPayload)
		log.Printf("%+v", srcPayload)
		require.NoError(err)
	})

	t.Run("Verify token", func(t *testing.T) {
		payload := PrincipalPayload{}
		gp, err := signer.ValidateToken(token, &payload)
		require.NoError(err)
		require.Equal(srcPayload, payload)
		require.Positive(gp.IssuedAt.Unix())
		require.Equal(testApp, gp.AppQName)
		require.Equal(testDuration, gp.Duration)
	})

}

func TestBasicUsage_BLOBUploadingPayload(t *testing.T) {

	require := require.New(t)
	signer := itokensjwt.TestTokensJWT()

	srcPayload := BLOBUploadingPayload{
		Workspace: istructs.WSID(1),
		BLOB:      istructs.RecordID(1),
		MaxSize:   20000,
	}

	var token string
	var err error

	t.Run("Prepare token", func(t *testing.T) {
		token, err = signer.IssueToken(testApp, testDuration, &srcPayload)
		log.Printf("%+v", srcPayload)
		require.NoError(err)
	})

	t.Run("Verify token", func(t *testing.T) {
		payload := BLOBUploadingPayload{}
		gp, err := signer.ValidateToken(token, &payload)
		require.NoError(err)
		require.Equal(srcPayload, payload)
		require.Positive(gp.IssuedAt.Unix())
		require.Equal(testApp, gp.AppQName)
		require.Equal(testDuration, gp.Duration)
	})
}

func TestBasicUsage_VerifiedValue(t *testing.T) {

	require := require.New(t)
	signer := itokensjwt.TestTokensJWT()
	testQName := appdef.NewQName("test", "entity")

	token := ""
	var err error

	t.Run("Issue token", func(t *testing.T) {
		payload := VerifiedValuePayload{
			VerificationKind: appdef.VerificationKind_EMail,
			WSID:             43,
			Entity:           testQName,
			Field:            "testName",
			Value:            42,
		}
		token, err = signer.IssueToken(testApp, testDuration, &payload)
		require.NoError(err)
	})

	t.Run("Verify token", func(t *testing.T) {
		payload := VerifiedValuePayload{}
		gp, err := signer.ValidateToken(token, &payload)
		require.NoError(err)
		require.Equal(appdef.VerificationKind_EMail, payload.VerificationKind)
		require.Equal(testQName, payload.Entity)
		require.Equal("testName", payload.Field)
		require.Equal(json.Number("42"), payload.Value)
		require.Positive(gp.IssuedAt.Unix())
		require.Equal(testApp, gp.AppQName)
		require.Equal(testDuration, gp.Duration)
		require.Equal(istructs.WSID(43), payload.WSID)
	})
}

func TestBasicUsage_IAppTokens(t *testing.T) {
	require := require.New(t)
	tokens := itokensjwt.TestTokensJWT()
	atf := ProvideIAppTokensFactory(tokens)
	at := atf.New(testApp)

	token := ""
	var err error

	t.Run("Issue token", func(t *testing.T) {
		srcPayload := PrincipalPayload{
			Login:       "login",
			SubjectKind: istructs.SubjectKind_User,
			ProfileWSID: istructs.WSID(10),
		}
		token, err = at.IssueToken(testDuration, &srcPayload)
		require.NoError(err)
	})

	t.Run("Validate token", func(t *testing.T) {
		payload := PrincipalPayload{}
		gp, err := at.ValidateToken(token, &payload)
		require.NoError(err)
		require.Positive(gp.IssuedAt.Unix())
		require.Equal(testApp, gp.AppQName)
		require.Equal(testDuration, gp.Duration)
	})

	t.Run("Basic validation error", func(t *testing.T) {
		testingu.MockTime.Add(testDuration * 2)
		defer func() { testingu.MockTime.Add(-testDuration * 2) }()
		payload := PrincipalPayload{}
		_, err := at.ValidateToken(token, &payload)
		require.ErrorIs(err, itokens.ErrTokenExpired)
	})

	t.Run("Error on validate a token issued for an another app", func(t *testing.T) {
		tokens := itokensjwt.TestTokensJWT()
		atf := ProvideIAppTokensFactory(tokens)
		at := atf.New(istructs.AppQName_test2_app1)
		payload := PrincipalPayload{}
		_, err := at.ValidateToken(token, &payload)
		require.Equal(ErrTokenIssuedForAnotherApp, err)
	})
}

func TestPrincipalPayloadRoleScopes(t *testing.T) {
	require := require.New(t)
	require.Zero(RoleScopeWorkspacesOwnedBy)
	require.Equal(1, int(RoleScopeWorkspace))

	cases := []struct {
		name  string
		api   bool
		roles []RoleType
	}{
		{name: "nil roles"},
		{name: "empty roles", roles: []RoleType{}},
		{name: "default scope", roles: []RoleType{{WSID: 100, QName: appdef.NewQName("test", "owner")}}},
		{name: "exact scopes", api: true, roles: []RoleType{
			{Scope: RoleScopeWorkspace, WSID: 200, QName: appdef.NewQName("test", "first")},
			{Scope: RoleScopeWorkspace, WSID: 100, QName: appdef.NewQName("test", "second")},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := PrincipalPayload{
				Login: "login", Alias: "alias", SubjectKind: istructs.SubjectKind_User,
				ProfileWSID: 42, GlobalRoles: []appdef.QName{appdef.NewQName("test", "global")},
				Roles: tc.roles, IsAPIToken: tc.api,
			}
			before := src
			before.Roles = slices.Clone(src.Roles)
			before.GlobalRoles = slices.Clone(src.GlobalRoles)
			data, err := json.Marshal(&src)
			require.NoError(err)
			var wire struct{ Roles []map[string]json.RawMessage }
			require.NoError(json.Unmarshal(data, &wire))
			require.Len(wire.Roles, len(src.Roles))
			for i, role := range src.Roles {
				require.Contains(wire.Roles[i], "Scope")
				var scope int
				require.NoError(json.Unmarshal(wire.Roles[i]["Scope"], &scope))
				require.Equal(int(role.Scope), scope)
			}
			var decoded PrincipalPayload
			require.NoError(json.Unmarshal(data, &decoded))
			require.Equal(src, decoded)
			tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, timeu.NewITime())
			appTokens := ProvideIAppTokensFactory(tokens).New(testApp)
			for _, path := range []string{"direct", "application-bound"} {
				t.Run(path, func(t *testing.T) {
					var token string
					var err error
					if path == "direct" {
						token, err = tokens.IssueToken(testApp, testDuration, &src)
					} else {
						token, err = appTokens.IssueToken(testDuration, &src)
					}
					require.NoError(err)
					var actual PrincipalPayload
					var gp istructs.GenericPayload
					if path == "direct" {
						gp, err = tokens.ValidateToken(token, &actual)
					} else {
						gp, err = appTokens.ValidateToken(token, &actual)
					}
					require.NoError(err)
					require.Equal(src, actual)
					require.Equal(testApp, gp.AppQName)
					require.Equal(testDuration, gp.Duration)
					parsed, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
					require.NoError(err)
					require.Equal("payloads.PrincipalPayload", parsed.Claims.(jwt.MapClaims)["aud"])
					require.Equal(before, src)
				})
			}
		})
	}
}

func TestPrincipalPayloadLegacyScopes(t *testing.T) {
	require := require.New(t)
	for _, api := range []bool{false, true} {
		name := "regular token"
		wantScope := RoleScopeWorkspacesOwnedBy
		if api {
			name, wantScope = "API token", RoleScopeWorkspace
		}
		t.Run(name, func(t *testing.T) {
			roles := []map[string]any{
				{"WSID": 100, "QName": "test.first"},
				{"WSID": 200, "QName": "test.second"},
			}
			tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, timeu.NewITime())
			appTokens := ProvideIAppTokensFactory(tokens).New(testApp)
			token := signPrincipalRoleClaims(t, require, api, roles)
			var actual PrincipalPayload
			gp, err := appTokens.ValidateToken(token, &actual)
			require.NoError(err)
			require.Equal([]RoleType{
				{Scope: wantScope, WSID: 100, QName: appdef.NewQName("test", "first")},
				{Scope: wantScope, WSID: 200, QName: appdef.NewQName("test", "second")},
			}, actual.Roles)
			require.Equal(api, actual.IsAPIToken)
			require.Equal("login", actual.Login)
			require.Equal("alias", actual.Alias)
			require.Equal(istructs.WSID(42), actual.ProfileWSID)
			require.Equal(testApp, gp.AppQName)
			require.Equal(testDuration, gp.Duration)

			t.Run("legacy reissue drops additive scopes", func(t *testing.T) {
				data, err := json.Marshal(&actual)
				require.NoError(err)
				var wire struct{ Roles []map[string]any }
				require.NoError(json.Unmarshal(data, &wire))
				for _, role := range wire.Roles {
					require.Contains(role, "Scope")
					delete(role, "Scope")
				}
				var reissued PrincipalPayload
				_, err = tokens.ValidateToken(signPrincipalRoleClaims(t, require, api, wire.Roles), &reissued)
				require.NoError(err)
				require.Equal(actual, reissued)
			})
		})
	}
}

func TestPrincipalPayloadExplicitScopes(t *testing.T) {
	require := require.New(t)
	for _, api := range []bool{false, true} {
		name := "regular token"
		wantDefault := RoleScopeWorkspacesOwnedBy
		if api {
			name, wantDefault = "API token", RoleScopeWorkspace
		}
		t.Run(name, func(t *testing.T) {
			token := signPrincipalRoleClaims(t, require, api, []map[string]any{
				{"WSID": 100, "QName": "test.role"},
				{"Scope": 0, "WSID": 100, "QName": "test.role"},
				{"Scope": 1, "WSID": 100, "QName": "test.role"},
			})
			tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, timeu.NewITime())
			var actual PrincipalPayload
			_, err := tokens.ValidateToken(token, &actual)
			require.NoError(err)
			require.Equal([]RoleType{
				{Scope: wantDefault, WSID: 100, QName: appdef.NewQName("test", "role")},
				{Scope: RoleScopeWorkspacesOwnedBy, WSID: 100, QName: appdef.NewQName("test", "role")},
				{Scope: RoleScopeWorkspace, WSID: 100, QName: appdef.NewQName("test", "role")},
			}, actual.Roles)
		})
	}
}

func TestPrincipalPayloadInvalidScopes(t *testing.T) {
	require := require.New(t)
	for _, tc := range []struct {
		name  string
		scope any
	}{
		{"null", nil}, {"string", "0"}, {"fraction", 0.5},
		{"negative", -1}, {"unknown", 2}, {"boolean", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := signPrincipalRoleClaims(t, require, false, []map[string]any{
				{"Scope": tc.scope, "WSID": 100, "QName": "test.role"},
			})
			tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, timeu.NewITime())
			var actual PrincipalPayload
			_, err := tokens.ValidateToken(token, &actual)
			require.Error(err)
			_, err = ProvideIAppTokensFactory(tokens).New(testApp).ValidateToken(token, &actual)
			require.Error(err)
		})
	}
	for name, scope := range map[string]RoleScope{"negative issued scope is rejected": -1, "unknown issued scope is rejected": 2} {
		t.Run(name, func(t *testing.T) {
			src := PrincipalPayload{Roles: []RoleType{{Scope: scope, WSID: 100, QName: appdef.NewQName("test", "role")}}}
			before := src
			before.Roles = slices.Clone(src.Roles)
			_, err := json.Marshal(&src)
			require.NoError(err)
			tokens := itokensjwt.ProvideITokens(itokensjwt.SecretKeyExample, timeu.NewITime())
			appTokens := ProvideIAppTokensFactory(tokens).New(testApp)
			token, err := appTokens.IssueToken(testDuration, &src)
			require.NoError(err)
			var actual PrincipalPayload
			_, err = tokens.ValidateToken(token, &actual)
			require.Error(err)
			_, err = appTokens.ValidateToken(token, &actual)
			require.Error(err)
			require.Equal(before, src)
		})
	}
}

func signPrincipalRoleClaims(t *testing.T, require *require.Assertions, api bool, roles []map[string]any) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"aud": "payloads.PrincipalPayload", "iat": now.Unix(), "exp": now.Add(testDuration).Unix(),
		"AppQName": &testApp, "Duration": testDuration, "IssuedAt": now,
		"Login": "login", "Alias": "alias", "SubjectKind": istructs.SubjectKind_User,
		"ProfileWSID": 42, "IsAPIToken": api, "Roles": roles,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(itokensjwt.SecretKeyExample))
	require.NoError(err)
	return token
}
