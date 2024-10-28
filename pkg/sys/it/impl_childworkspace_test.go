/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
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
		resp := vit.PostWS(parentWS, "q.sys.QueryChildWorkspaceByName", body, coreutils.Expect404())
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
			resp := vit.PostWS(parentWS, "c.sys.InitChildWorkspace", body, coreutils.Expect409())
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
		vit.PostWS(parentWS, "c.sys.CUD", cudBody, coreutils.Expect403(), coreutils.WithAuthorizeBy(newPrn.Token))

		// make this new foreign login a subject in the existing workspace
		body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "sys.Subject","Login": "%s","SubjectKind":%d,"Roles": "%s","ProfileWSID":%d}}]}`,
			newLoginName, istructs.SubjectKind_User, iauthnz.QNameRoleWorkspaceOwner, newPrn.ProfileWSID)
		vit.PostWS(parentWS, "c.sys.CUD", body)

		// now the foreign login could work in the workspace
		vit.PostWS(parentWS, "c.sys.CUD", cudBody, coreutils.WithAuthorizeBy(newPrn.Token))
	})

	t.Run("enrich principal token", func(t *testing.T) {
		// 403 forbidden on try to execute a stricted operation in the child workspace using the non-enriched token
		body = `{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "app1pkg.articles","name": "cola","article_manual": 1,"article_hash": 2,"hideonhold": 3,"time_active": 4,"control_active": 5}}]}`
		vit.PostWS(childWS, "c.sys.CUD", body, coreutils.Expect403())

		// create cdoc.sys.Subject with a role the custom func execution could be authorized with
		body = fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "sys.Subject","Login": "login","SubjectKind":%d,"Roles": "%s","ProfileWSID":%d}}]}`,
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
		vit.PostWS(childWS, "c.sys.CUD", body, coreutils.WithAuthorizeBy(enrichedToken))
	})

	t.Run("API token", func(t *testing.T) {
		// 403 forbidden on try to execute a stricted operation in the child workspace
		body = `{"args":{"Schema":"app1pkg.articles"},"elements":[{"fields":["sys.ID"]}]}`
		vit.PostWS(childWS, "q.sys.Collection", body, coreutils.Expect403())

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
		vit.PostWS(childWS, "q.sys.Collection", body, coreutils.WithAuthorizeBy(apiToken))
	})
}
