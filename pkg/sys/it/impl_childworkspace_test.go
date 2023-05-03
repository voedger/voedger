/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iauthnzimpl"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_ChildWorkspaces(t *testing.T) {
	require := require.New(t)
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	parentWS := hit.WS(istructs.AppQName_test1_app1, "test_ws")
	wsName := hit.NextName()

	t.Run("404 not found on unexisting child workspace query", func(t *testing.T) {
		body := fmt.Sprintf(`
			{
				"args": {
					"WSName": "%s"
				},
				"elements":[
					{
						"fields":["WSID"]
					}
				]
			}`, wsName)
		resp := hit.PostWS(parentWS, "q.sys.QueryChildWorkspaceByName", body, utils.Expect404())
		resp.Println()
	})

	t.Run("create child workspace", func(t *testing.T) {
		// init
		body := fmt.Sprintf(`
			{
				"args": {
					"WSName": "%s",
					"WSKind": "my.WSKind",
					"WSKindInitializationData": "{\"IntFld\": 10}",
					"TemplateName": "test_template",
					"WSClusterID": 42
				}
			}`, wsName)
		hit.PostWS(parentWS, "c.sys.InitChildWorkspace", body)

		// wait for finish
		childWS := hit.WaitForChildWorkspace(parentWS, wsName, parentWS.Owner)
		require.Empty(childWS.WSError)
		require.Equal(wsName, childWS.Name)
		require.Equal(it.QNameTestWSKind, childWS.Kind)
		require.Equal(`{"IntFld": 10}`, childWS.InitDataJSON)
		require.Equal("test_template", childWS.TemplateName)
		require.Equal(istructs.ClusterID(42), childWS.WSID.ClusterID())

		t.Run("create a new workspace with an existing name -> 409 conflict", func(t *testing.T) {
			body := fmt.Sprintf(`{"args": {"WSName": "%s","WSKind": "my.WSKind","WSKindInitializationData": "{\"WorkStartTime\": \"10\"}","TemplateName": "test","WSClusterID": 1}}`, wsName)
			resp := hit.PostWS(parentWS, "c.sys.InitChildWorkspace", body, utils.Expect409())
			resp.Println()
		})
	})

	t.Run("read child workspaces list", func(t *testing.T) {
		body := `{"args":{"Schema":"sys.ChildWorkspace"},"elements":[{"fields":["WSName","WSKind","WSID","WSError"]}]}`
		resp := hit.PostWS(parentWS, "q.sys.Collection", body)
		// note: wsKind is rendered as {} because q.sys.Collection appends QName to the object to marshal to JSON by value
		// whereas appdef.QName.MarshalJSON() func has pointer receiver
		resp.Println()
	})
}

func TestForeignAuthorization(t *testing.T) {
	require := require.New(t)
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	// sign up a new login
	newLoginName := hit.NextName()
	newLogin := hit.SignUp(newLoginName, "1", istructs.AppQName_test1_app1)
	newPrn := hit.SignIn(newLogin)

	parentWS := hit.WS(istructs.AppQName_test1_app1, "test_ws")
	wsName := hit.NextName()

	// init child workspace
	body := fmt.Sprintf(`{"args": {"WSName": "%s","WSKind": "my.WSKind","WSKindInitializationData": "{\"IntFld\": 10}","TemplateName": "test_template","WSClusterID": 42}}`, wsName)
	hit.PostWS(parentWS, "c.sys.InitChildWorkspace", body)

	// wait for finish
	childWS := hit.WaitForChildWorkspace(parentWS, wsName, parentWS.Owner)

	t.Run("subjects", func(t *testing.T) {
		// try to execute an operation by the foreign login, expect 403
		cudBody := `{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "untill.articles","name": "cola","article_manual": 1,"article_hash": 2,"hideonhold": 3,"time_active": 4,"control_active": 5}}]}`
		hit.PostWS(parentWS, "c.sys.CUD", cudBody, utils.Expect403(), coreutils.WithAuthorizeBy(newPrn.Token))

		// make this new foreign login a subject in the existing workspace
		body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "sys.Subject","Login": "%s","SubjectKind":%d,"Roles": "%s"}}]}`,
			newLoginName, istructs.SubjectKind_User, iauthnz.QNameRoleWorkspaceSubject)
		hit.PostWS(parentWS, "c.sys.CUD", body)

		// now the foreign login could work in the workspace
		hit.PostWS(parentWS, "c.sys.CUD", cudBody, coreutils.WithAuthorizeBy(newPrn.Token))
	})

	t.Run("enrich principal token", func(t *testing.T) {
		// 403 forbidden on try to execute a stricted operation in the child workspace using the non-enriched token
		body = `{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "untill.articles","name": "cola","article_manual": 1,"article_hash": 2,"hideonhold": 3,"time_active": 4,"control_active": 5}}]}`
		hit.PostWS(childWS, "c.sys.CUD", body, utils.Expect403())

		// create cdoc.sys.Subject with a role the custom func execution could be authorized with
		body = fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "sys.Subject","Login": "login","SubjectKind":%d,"Roles": "%s"}}]}`,
			istructs.SubjectKind_User, iauthnz.QNameRoleWorkspaceSubject)
		hit.PostWS(parentWS, "c.sys.CUD", body)

		// enrich the principal token in the parentWS
		// basic auth
		body = `{"args":{"Login":"login"},"elements":[{"fields":["EnrichedToken"]}]}`
		resp := hit.PostWS(parentWS, "q.sys.EnrichPrincipalToken", body)
		enrichedToken := resp.SectionRow()[0].(string)

		// ok to execute a stricted operation in the child workspace using the enriched token
		// expect no errors
		body = `{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "untill.articles","name": "cola","article_manual": 1,"article_hash": 2,"hideonhold": 3,"time_active": 4,"control_active": 5}}]}`
		hit.PostWS(childWS, "c.sys.CUD", body, coreutils.WithAuthorizeBy(enrichedToken))
	})

	t.Run("API token", func(t *testing.T) {
		// 403 forbidden on try to execute a stricted operation in the child workspace
		body = `{"args":{"Schema":"untill.articles"},"elements":[{"fields":["sys.ID"]}]}`
		hit.PostWS(childWS, "q.sys.Collection", body, utils.Expect403())

		// issue an API token
		as, err := hit.IAppStructsProvider.AppStructs(istructs.AppQName_test1_app1)
		require.NoError(err)
		apiToken, err := iauthnzimpl.IssueAPIToken(as.AppTokens(), time.Hour, []appdef.QName{appdef.NewQName("air", "AirReseller")}, childWS.WSID, payloads.PrincipalPayload{
			Login:       parentWS.Owner.Name,
			SubjectKind: istructs.SubjectKind_User,
			ProfileWSID: parentWS.Owner.ProfileWSID,
		})
		require.NoError(err)

		// API token has role.air.AirReseller, q.sys.Collection is allowed for that role according to the current ACL -> the request should be successful
		hit.PostWS(childWS, "q.sys.Collection", body, coreutils.WithAuthorizeBy(apiToken))
	})
}
