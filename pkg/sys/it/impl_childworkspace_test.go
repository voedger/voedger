/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iauthnzimpl"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_ChildWorkspaces(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	parentWS := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	wsName := vit.NextName()

	t.Run("404 not found on unexisting child workspace query", func(t *testing.T) {
		body := fmt.Sprintf(`
			{
				"args": {
					"WSName": %q
				},
				"elements":[
					{
						"fields":["WSID"]
					}
				]
			}`, wsName)
		resp := vit.PostWS(parentWS, "q.sys.QueryChildWorkspaceByName", body, httpu.Expect404())
		resp.Println()
	})

	t.Run("create child workspace", func(t *testing.T) {
		// init
		// note: creating workspace at non-main cluster is unsupported for now
		// because there are no AppWorkspaces in any cluster but Main (created automatically on VVM launch)
		// also GetAppWSID() uses MainCluser, not the target one
		body := fmt.Sprintf(`
			{
				"args": {
					"WSName": %q,
					"WSKind": "app1pkg.test_ws",
					"WSKindInitializationData": "{\"IntFld\": 10}",
					"TemplateName": "test_template",
					"WSClusterID": 1
				}
			}`, wsName)
		vit.PostWS(parentWS, "c.sys.InitChildWorkspace", body)

		// wait for finish
		childWS := vit.WaitForChildWorkspace(parentWS, wsName)
		require.Empty(childWS.WSError)
		require.Equal(wsName, childWS.Name)
		require.Equal(it.QNameApp1_TestWSKind, childWS.Kind)
		require.Equal(`{"IntFld": 10}`, childWS.InitDataJSON)
		require.Equal("test_template", childWS.TemplateName)
		require.Equal(istructs.ClusterID(1), childWS.WSID.ClusterID())

		t.Run("create a new workspace with an existing name -> 409 conflict", func(t *testing.T) {
			body := fmt.Sprintf(`{"args": {"WSName": %q,"WSKind": "app1pkg.test_ws","WSKindInitializationData": "{\"WorkStartTime\": \"10\"}","TemplateName": "test","WSClusterID": 1}}`, wsName)
			resp := vit.PostWS(parentWS, "c.sys.InitChildWorkspace", body, httpu.Expect409())
			resp.Println()
		})
	})

	t.Run("read child workspaces list", func(t *testing.T) {
		body := `{"args":{"Schema":"sys.ChildWorkspace"},"elements":[{"fields":["WSName","WSKind","WSID","WSError"]}]}`
		resp := vit.PostWS(parentWS, "q.sys.Collection", body)
		// note: wsKind is rendered as {} because q.sys.Collection appends QName to the object to marshal to JSON by value
		// whereas appdef.QName.MarshalJSON() func has pointer receiver
		resp.Println()
	})
}

func TestForeignAuthorization(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// sign up a new login
	newLoginName := vit.NextName()
	newLogin := vit.SignUp(newLoginName, "1", istructs.AppQName_test1_app1)
	newPrn := vit.SignIn(newLogin)

	parentWS := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	wsName := vit.NextName()

	// init child workspace
	body := fmt.Sprintf(`{"args": {"WSName": %q,"WSKind": "app1pkg.test_ws","WSKindInitializationData": "{\"IntFld\": 10}","TemplateName": "test_template","WSClusterID": 42}}`, wsName)
	vit.PostWS(parentWS, "c.sys.InitChildWorkspace", body)

	// wait for finish
	childWS := vit.WaitForChildWorkspace(parentWS, wsName)

	t.Run("subjects", func(t *testing.T) {
		// try to execute an operation by the foreign login, expect 403
		cudBody := `{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "app1pkg.articles","name": "cola","article_manual": 1,"article_hash": 2,"hideonhold": 3,"time_active": 4,"control_active": 5}}]}`
		vit.PostWS(parentWS, "c.sys.CUD", cudBody, httpu.Expect403(), httpu.WithAuthorizeBy(newPrn.Token))

		// make this new foreign login a subject in the existing workspace
		body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "sys.Subject","Login": %q,"SubjectKind":%d,"Roles": %q,"ProfileWSID":%d}}]}`,
			newLoginName, istructs.SubjectKind_User, iauthnz.QNameRoleWorkspaceOwner, newPrn.ProfileWSID)
		cdocSubjectID := vit.PostWS(parentWS, "c.sys.CUD", body).NewID()

		// now the foreign login could work in the workspace
		vit.PostWS(parentWS, "c.sys.CUD", cudBody, httpu.WithAuthorizeBy(newPrn.Token))

		// deactivate cdoc.Subject
		body = fmt.Sprintf(`{"cuds": [{"sys.ID": %d,"fields": {"sys.IsActive": false}}]}`, cdocSubjectID)
		vit.PostWS(parentWS, "c.sys.CUD", body)

		// the foreign login can no longer work in the workspace
		vit.PostWS(parentWS, "c.sys.CUD", cudBody, httpu.WithAuthorizeBy(newPrn.Token), httpu.Expect403())
	})

	t.Run("enrich principal token", func(t *testing.T) {
		// 403 forbidden on try to execute a stricted operation in the child workspace using the non-enriched token
		body = `{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "app1pkg.articles","name": "cola","article_manual": 1,"article_hash": 2,"hideonhold": 3,"time_active": 4,"control_active": 5}}]}`
		vit.PostWS(childWS, "c.sys.CUD", body, httpu.Expect403())

		// create cdoc.sys.Subject with a role the custom func execution could be authorized with
		body = fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "sys.Subject","Login": "login","SubjectKind":%d,"Roles": %q,"ProfileWSID":%d}}]}`,
			istructs.SubjectKind_User, iauthnz.QNameRoleWorkspaceOwner, newPrn.ProfileWSID)
		vit.PostWS(parentWS, "c.sys.CUD", body)

		// enrich the principal token in the parentWS
		// basic auth
		body = `{"args":{"Login":"login"},"elements":[{"fields":["EnrichedToken"]}]}`
		resp := vit.PostWS(parentWS, "q.sys.EnrichPrincipalToken", body)
		enrichedToken := resp.SectionRow()[0].(string)

		// ok to execute a stricted operation in the child workspace using the enriched token
		// expect no errors
		body = `{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "app1pkg.articles","name": "cola","article_manual": 1,"article_hash": 2,"hideonhold": 3,"time_active": 4,"control_active": 5}}]}`
		vit.PostWS(childWS, "c.sys.CUD", body, httpu.WithAuthorizeBy(enrichedToken))
		enrichedPrincipalPayload, enrichedGenericPayload := decodeScopedPrincipalToken(t, require, vit, enrichedToken)
		require.Contains(enrichedPrincipalPayload.Roles, payloads.RoleType{
			Scope: payloads.RoleScopeWorkspacesOwnedBy, WSID: parentWS.WSID, QName: iauthnz.QNameRoleWorkspaceOwner,
		})
		for _, role := range enrichedPrincipalPayload.Roles {
			require.Equal(payloads.RoleScopeWorkspacesOwnedBy, role.Scope)
		}

		enrichBody := `{"args":{"Login":"login"},"elements":[{"fields":["EnrichedToken"]}]}`
		repeatedToken := vit.PostWS(parentWS, "q.sys.EnrichPrincipalToken", enrichBody,
			httpu.WithAuthorizeBy(enrichedToken)).SectionRow()[0].(string)
		repeatedPayload, _ := decodeScopedPrincipalToken(t, require, vit, repeatedToken)
		require.Equal(enrichedPrincipalPayload, repeatedPayload)

		refreshBody := `{"args":{},"elements":[{"fields":["NewPrincipalToken"]}]}`
		refreshedToken := vit.PostProfile(parentWS.Owner, "q.sys.RefreshPrincipalToken", refreshBody,
			httpu.WithAuthorizeBy(repeatedToken)).SectionRow()[0].(string)
		refreshedPayload, refreshedMetadata := decodeScopedPrincipalToken(t, require, vit, refreshedToken)
		require.Equal(repeatedPayload, refreshedPayload)
		require.Equal(enrichedGenericPayload.AppQName, refreshedMetadata.AppQName)
		require.Equal(enrichedGenericPayload.Duration, refreshedMetadata.Duration)
		vit.PostWS(childWS, "c.sys.CUD", body, httpu.WithAuthorizeBy(refreshedToken))

		grandchildName := vit.NextName()
		grandchildBody, err := json.Marshal(map[string]any{"args": map[string]any{
			"WSName": grandchildName, "WSKind": "app1pkg.test_ws",
			"WSKindInitializationData": `{"IntFld":10}`, "WSClusterID": 1,
		}})
		require.NoError(err)
		vit.PostWS(childWS, "c.sys.InitChildWorkspace", string(grandchildBody), httpu.WithAuthorizeBy(refreshedToken))
		grandchildWS := vit.WaitForChildWorkspace(childWS, grandchildName, httpu.WithAuthorizeBy(refreshedToken))
		vit.PostWS(grandchildWS, "c.sys.CUD", body, httpu.WithAuthorizeBy(refreshedToken), httpu.Expect403())
		nextToken := vit.PostWS(childWS, "q.sys.EnrichPrincipalToken", enrichBody,
			httpu.WithAuthorizeBy(refreshedToken)).SectionRow()[0].(string)
		nextPayload, _ := decodeScopedPrincipalToken(t, require, vit, nextToken)
		require.Subset(nextPayload.Roles, refreshedPayload.Roles)
		require.Contains(nextPayload.Roles, payloads.RoleType{
			Scope: payloads.RoleScopeWorkspacesOwnedBy, WSID: childWS.WSID, QName: iauthnz.QNameRoleWorkspaceOwner,
		})
		vit.PostWS(grandchildWS, "c.sys.CUD", body, httpu.WithAuthorizeBy(nextToken))
	})

	t.Run("API token", func(t *testing.T) {
		// 403 forbidden on try to execute a stricted operation in the child workspace
		body = `{"args":{"Schema":"app1pkg.articles"},"elements":[{"fields":["sys.ID"]}]}`
		vit.PostWS(childWS, "q.sys.Collection", body, httpu.Expect403())

		// issue an API token
		as, err := vit.IAppStructsProvider.BuiltIn(istructs.AppQName_test1_app1)
		require.NoError(err)
		apiToken, err := iauthnzimpl.IssueAPIToken(as.AppTokens(), time.Hour, []appdef.QName{appdef.NewQName("app1pkg", "SpecialAPITokenRole")},
			childWS.WSID, payloads.PrincipalPayload{
				Login:       parentWS.Owner.Name,
				SubjectKind: istructs.SubjectKind_User,
				ProfileWSID: parentWS.Owner.ProfileWSID,
			})
		require.NoError(err)

		// API token has role.app1pkg.SpecialAPITokenRole, q.sys.Collection is allowed for that role according to grants in vsql -> the request should be successful
		vit.PostWS(childWS, "q.sys.Collection", body, httpu.WithAuthorizeBy(apiToken))
		apiPayload, _ := decodeScopedPrincipalToken(t, require, vit, apiToken)
		require.Equal([]payloads.RoleType{{
			Scope: payloads.RoleScopeWorkspace, WSID: childWS.WSID, QName: appdef.NewQName("app1pkg", "SpecialAPITokenRole"),
		}}, apiPayload.Roles)
		vit.PostWS(parentWS, "q.sys.Collection", body, httpu.WithAuthorizeBy(apiToken), httpu.Expect403())
	})
}

func TestRoleScopedToOneForeignWorkspace(t *testing.T) {
	require := require.New(t)

	// Workspace hierarchy and q.sys.Collection results with the foreign principal's scoped token:
	//
	// parentWS (test_ws)                     -> 403
	// +-- workspaces[0]                      -> 403
	// +-- workspaces[1] (targetWS)           -> 200 (WorkspaceOwner scoped to targetWS.WSID)
	// +-- workspaces[2]                      -> 403

	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	login := vit.SignUp(vit.NextName(), "1", istructs.AppQName_test1_app1)
	newPrincipal := vit.SignIn(login)
	parentWS := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	workspaces := make([]*it.AppWorkspace, 3)

	// create 3 childs of test_ws
	for i := range workspaces {
		name := vit.NextName()
		body := fmt.Sprintf(`{"args":{"WSName":%q,"WSKind":"app1pkg.test_ws","WSKindInitializationData":"{\"IntFld\":10}","TemplateName":"test_template","WSClusterID":1}}`, name)
		vit.PostWS(parentWS, "c.sys.InitChildWorkspace", body)
		workspaces[i] = vit.WaitForChildWorkspace(parentWS, name)
		require.Empty(workspaces[i].WSError)
	}
	targetWS := workspaces[1]
	body := `{"args":{"Schema":"app1pkg.articles"},"elements":[{"fields":["sys.ID"]}]}`

	// The foreign caller has no access before receiving the scoped grant.
	for _, ws := range []*it.AppWorkspace{parentWS, workspaces[0], targetWS, workspaces[2]} {
		vit.PostWS(ws, "q.sys.Collection", body, httpu.WithAuthorizeBy(newPrincipal.Token), httpu.Expect403())
	}

	// add role.sys.WorkspaceOwner to the actual token scoped to targetWorkspace
	as, err := vit.IAppStructsProvider.BuiltIn(istructs.AppQName_test1_app1)
	require.NoError(err)
	var expectedPayload payloads.PrincipalPayload
	_, err = as.AppTokens().ValidateToken(newPrincipal.Token, &expectedPayload)
	require.NoError(err)
	require.Empty(expectedPayload.Roles)
	require.Empty(expectedPayload.GlobalRoles)
	require.False(expectedPayload.IsAPIToken)
	expectedPayload.Roles = []payloads.RoleType{{
		Scope: payloads.RoleScopeWorkspace, WSID: targetWS.WSID, QName: iauthnz.QNameRoleWorkspaceOwner,
	}}
	token, err := as.AppTokens().IssueToken(time.Hour, &expectedPayload)
	require.NoError(err)
	actualPayload, _ := decodeScopedPrincipalToken(t, require, vit, token)
	require.Equal(expectedPayload, actualPayload)

	// now the enriched token allows to work in targetWS
	vit.PostWS(targetWS, "q.sys.Collection", body, httpu.WithAuthorizeBy(token))

	// but still unable to work in other foreign workspaces because it is scoped to targetWS only
	for _, ws := range []*it.AppWorkspace{workspaces[0], workspaces[2], parentWS} {
		vit.PostWS(ws, "q.sys.Collection", body, httpu.WithAuthorizeBy(token), httpu.Expect403())
	}
}

func decodeScopedPrincipalToken(t *testing.T, require *require.Assertions, vit *it.VIT, token string) (payloads.PrincipalPayload, istructs.GenericPayload) {
	t.Helper()
	var payload payloads.PrincipalPayload
	metadata, err := vit.ValidateToken(token, &payload)
	require.NoError(err)
	parsed, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
	require.NoError(err)
	roles := parsed.Claims.(jwt.MapClaims)["Roles"].([]any)
	require.Len(roles, len(payload.Roles))
	for i, role := range payload.Roles {
		claims := roles[i].(map[string]any)
		require.Contains(claims, "Scope")
		require.Equal(float64(role.Scope), claims["Scope"])
	}
	return payload, metadata
}
