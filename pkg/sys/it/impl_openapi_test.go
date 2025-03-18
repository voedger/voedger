/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package sys_it

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/acl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/processors/query2/openapi"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestOpenAPI(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	require := require.New(t)
	//ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	appDef, err := vit.AppDef(istructs.AppQName_test1_app1)
	ws := appDef.Workspace(appdef.NewQName("app1pkg", "test_wsWS"))
	require.NotNil(ws)
	require.NoError(err)

	writer := new(bytes.Buffer)

	schemaMeta := openapi.SchemaMeta{
		SchemaTitle:   "Test Schema",
		SchemaVersion: "1.0.0",
		AppName:       appdef.NewAppQName("voedger", "testapp"),
	}

	err = openapi.CreateOpenApiSchema(writer, ws, appdef.NewQName("app1pkg", "ApiRole"), acl.PublishedTypes, schemaMeta)

	require.NoError(err)

	json := writer.String()
	require.Contains(json, "\"components\": {")
	require.Contains(json, "\"app1pkg.Currency\": {")
	require.Contains(json, "\"paths\": {")
	require.Contains(json, "/api/v2/users/voedger/apps/testapp/workspaces/{wsid}/docs/app1pkg.Currency")

	//ioutil.WriteFile("schema.json", writer.Bytes(), 0644)
	//require.True(false)
}
