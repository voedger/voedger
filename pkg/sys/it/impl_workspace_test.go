/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package heeus_it

import (
	"fmt"
	"log"
	"strconv"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	wsuntill "github.com/untillpro/airs-bp3/packages/air/workspace"
	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/authnz/workspace"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
)

func TestBasicUsage_Workspace(t *testing.T) {
	require := require.New(t)
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	loginName := hit.NextName()
	wsName := hit.NextName()
	login := hit.SignUp(loginName, "1", istructs.AppQName_test1_app1)
	prn := hit.SignIn(login)

	t.Run("404 not found on q.sys.QueryChildWorkspaceByName on a non-inited workspace", func(t *testing.T) {
		body := fmt.Sprintf(`
			{
				"args": {
					"WSName": "%s"
				},
				"elements":[
					{
						"fields":["WSName", "WSKind", "WSKindInitializationData", "TemplateName", "TemplateParams", "WSID", "WSError"]
					}
				]
			}`, wsName)
		resp := hit.PostProfile(prn, "q.sys.QueryChildWorkspaceByName", body, utils.Expect404())
		resp.Println()
	})

	t.Run("init user workspace (create a restaurant)", func(t *testing.T) {
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
		hit.PostProfile(prn, "c.sys.InitChildWorkspace", body)
		ws := hit.WaitForWorkspace(wsName, prn)

		require.Empty(ws.WSError)
		require.Equal(wsName, ws.Name)
		require.Equal(it.QNameTestWSKind, ws.Kind)
		require.Equal(`{"IntFld": 10}`, ws.InitDataJSON)
		require.Equal("test_template", ws.TemplateName)
		require.Equal(istructs.ClusterID(42), ws.WSID.ClusterID())

		t.Run("check the initialized workspace using collection", func(t *testing.T) {
			body = `{"args":{"Schema":"untill.air_table_plan"},"elements":[{"fields":["sys.ID","image","preview"]}]}`
			resp := hit.PostWS(ws, "q.sys.Collection", body)
			require.Equal(int64(5000000000400), int64(resp.SectionRow()[0].(float64)))  // from testTemplate
			require.Equal(int64(5000000000416), int64(resp.SectionRow(1)[0].(float64))) // from testTemplate
		})

		var idOfCDocWSKind int64

		t.Run("check current cdoc.sys.$wsKind", func(t *testing.T) {
			cdoc, id := hit.GetCDocWSKind(ws)
			idOfCDocWSKind = id
			require.Equal(float64(10), cdoc["IntFld"])
			require.Equal("", cdoc["StrFld"])
			require.Len(cdoc, 2)
		})

		t.Run("reconfigure the workspace", func(t *testing.T) {
			// CDoc<my.wsKind> is a singleton
			body = fmt.Sprintf(`
				{
					"cuds": [
						{
							"sys.ID": %d,
							"fields": {
								"sys.QName": "my.WSKind",
								"IntFld": 42,
								"StrFld": "str"
							}
						}
					]
				}`, idOfCDocWSKind)
			hit.PostWS(ws, "c.sys.CUD", body)

			// check updated workspace config
			cdoc, _ := hit.GetCDocWSKind(ws)
			require.Equal(2, len(cdoc))
			require.Equal(float64(42), cdoc["IntFld"])
			require.Equal("str", cdoc["StrFld"])
		})
	})

	t.Run("create a new workspace with an existing name -> 409 conflict", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"WSName": "%s","WSKind": "my.WSKind","WSKindInitializationData": "{\"WorkStartTime\": \"10\"}","TemplateName": "test","WSClusterID": 1}}`, wsName)
		resp := hit.PostProfile(prn, "c.sys.InitChildWorkspace", body, utils.Expect409())
		resp.Println()
	})

	t.Run("read user workspaces list", func(t *testing.T) {
		body := `{"args":{"Schema":"sys.ChildWorkspace"},"elements":[{"fields":["WSName","WSKind","WSID","WSError"]}]}`
		resp := hit.PostProfile(prn, "q.sys.Collection", body)
		// note: wsKind is rendered as {} because q.sys.Collection appends QName to the object to marshal to JSON by value
		// whereas appdef.QName.MarshalJSON() func has pointer receiver
		resp.Println()
	})
}

func TestWorkspaceAuthorization(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	ws := hit.WS(istructs.AppQName_test1_app1, "test_ws")
	prn := ws.Owner

	body := `{"cuds": [{"sys.ID": 1,"fields": {"sys.QName": "my.WSKind"}}]}`

	t.Run("403 forbidden", func(t *testing.T) {
		t.Run("workspace is not initialized", func(t *testing.T) {
			// try to exec c.sys.CUD in non-inited ws id 1
			hit.PostApp(istructs.AppQName_test1_app1, 1, "c.sys.CUD", body, utils.WithAuthorizeBy(prn.Token), utils.Expect403()).Println()
		})

		t.Run("access denied (wrong wsid)", func(t *testing.T) {
			// create a new login
			login := hit.SignUp(hit.NextName(), "1", istructs.AppQName_test1_app1)
			newPrn := hit.SignIn(login)

			// try to modify the workspace by the non-owner
			hit.PostApp(istructs.AppQName_test1_app1, ws.WSID, "c.sys.CUD", body, utils.WithAuthorizeBy(newPrn.Token), utils.Expect403()).Println()
		})
	})

	t.Run("401 unauthorized", func(t *testing.T) {
		t.Run("token from an another app", func(t *testing.T) {
			login := hit.SignUp(hit.NextName(), "1", istructs.AppQName_test1_app2)
			newPrn := hit.SignIn(login)
			hit.PostApp(istructs.AppQName_test1_app1, ws.WSID, "c.sys.CUD", body, utils.WithAuthorizeBy(newPrn.Token), utils.Expect401()).Println()
		})
	})
}

// TODO: shoot off air.Restaurant check to airs-bp3 integration tests
func TestDenyCreateCDocWSKind(t *testing.T) {
	cdocWSKinds := []appdef.QName{
		authnz.QNameCDoc_WorkspaceKind_UserProfile,
		authnz.QNameCDoc_WorkspaceKind_DeviceProfile,
		wsuntill.QNameCDocWorkspaceKindRestaurant,
		authnz.QNameCDoc_WorkspaceKind_AppWorkspace,
	}

	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	ws := hit.WS(istructs.AppQName_test1_app1, "test_ws")

	for _, cdocWSkind := range cdocWSKinds {
		t.Run("deny to create manually cdoc.sys."+cdocWSkind.String(), func(t *testing.T) {
			body := fmt.Sprintf(`{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "%s"}}]}`, cdocWSkind.String())
			hit.PostWS(ws, "c.sys.CUD", body, utils.Expect403()).Println()
		})
	}
}

func TestDenyCUDCDocOwnerModification(t *testing.T) {
	hit := it.NewHIT(t, &it.SharedConfig_Simple)
	defer hit.TearDown()

	ws := hit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("CDoc<ChildWorkspace>", func(t *testing.T) {
		// try to modify CDoc<ChildWorkspace>
		_, idOfCDocWSKind := hit.GetCDocChildWorkspace(ws)
		body := fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"WSName":"new name"}}]}`, idOfCDocWSKind) // intFld is declared in hit.SharedConfig_Simple
		hit.PostProfile(ws.Owner, "c.sys.CUD", body, utils.Expect403()).Println()
	})

	// note: unable to work with CDoc<Login>
}

func TestWorkspaceTemplatesValidationErrors(t *testing.T) {
	dummyFile := &fstest.MapFile{}
	cases := []struct {
		desc       string
		blobs      []string
		noDataFile bool
		data       string
		wsInitData string
	}{
		{desc: "no data file", noDataFile: true},
		{desc: "malformed JSON in data file", data: "wrong"},
		{desc: "blob fn format: no ID", blobs: []string{"image.png"}},
		{desc: "blob fn format: no ID", blobs: []string{"_image.png"}},
		{desc: "blob fn format: no blob field", blobs: []string{"42.png"}},
		{desc: "blob fn format: no blob field", blobs: []string{"42_.png"}},
		{desc: "blob fn format: wrong ID", blobs: []string{"sdf_image.png"}},
		{desc: "unknown blob field", blobs: []string{"42_unknown.png"}, data: `[{"sys.ID":42}]`},
		{desc: "orphaned blob", blobs: []string{"43_image.png"}, data: `[{"sys.ID":42}]`},
		{desc: "duplicate blob", blobs: []string{"42_image.png", "42_image.jpg"}, data: `[{"sys.ID":42}]`},
		{desc: "record with no sys.ID field in data file", data: `[{"sys.IsActive":true}]`},

		// TODO: the following was tested by waiting for the workspace error. Find out how to test it quickly
		// {desc: "invalid wsKindInitializationData", data: `[{"sys.ID":42}]`, wsInitData: `wrong`},
		// {desc: "unknown field in wsKindInitializationData", data: `[{"sys.ID":42}]`, wsInitData: `{"unknown": "10"}`},
		// {desc: "unsupported data type in wsKindInitializationData", data: `[{"sys.ID":42}]`, wsInitData: `{"IntFld": {}}`},
	}

	epWSTemplates := vvm.NewRootExtensionPoint()
	epTestWSKindTemplates := epWSTemplates.ExtensionPoint(it.QNameTestWSKind)
	for i, c := range cases {
		str := strconv.Itoa(i)
		fs := fstest.MapFS{}
		if len(c.data) > 0 {
			fs["data.json"] = &fstest.MapFile{Data: []byte(c.data)}
		} else if !c.noDataFile {
			fs["data.json"] = &fstest.MapFile{Data: []byte("[]")}
		}
		for _, df := range c.blobs {
			fs[df] = dummyFile
		}
		epTestWSKindTemplates.AddNamed("test"+str, fs)
	}

	for i, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			fs := fstest.MapFS{}
			if len(c.data) > 0 {
				fs["data.json"] = &fstest.MapFile{Data: []byte(c.data)}
			} else if !c.noDataFile {
				fs["data.json"] = &fstest.MapFile{Data: []byte("[]")}
			}
			for _, df := range c.blobs {
				fs[df] = dummyFile
			}
			str := strconv.Itoa(i)
			_, _, err := workspace.ValidateTemplate("test"+str, epTestWSKindTemplates, it.QNameTestWSKind)
			require.NotNil(t, err)
			log.Println(err)
		})
	}

	t.Run("no template for workspace kind", func(t *testing.T) {
		_, _, err := workspace.ValidateTemplate("test", epTestWSKindTemplates, appdef.NewQName("sys", "unknownKind"))
		require.NotNil(t, err)
		log.Println(err)
	})
}
