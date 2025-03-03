/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"log"
	"strconv"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/workspace"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_Workspace(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	loginName := vit.NextName()
	wsName := vit.NextName()
	login := vit.SignUp(loginName, "1", istructs.AppQName_test1_app1)
	prn := vit.SignIn(login)

	t.Run("404 not found on q.sys.QueryChildWorkspaceByName on a non-inited workspace", func(t *testing.T) {
		body := fmt.Sprintf(`
			{
				"args": {
					"WSName": %q
				},
				"elements":[
					{
						"fields":["WSName", "WSKind", "WSKindInitializationData", "TemplateName", "TemplateParams", "WSID", "WSError"]
					}
				]
			}`, wsName)
		resp := vit.PostProfile(prn, "q.sys.QueryChildWorkspaceByName", body, coreutils.Expect404())
		resp.Println()
	})

	t.Run("init user workspace (create a restaurant)", func(t *testing.T) {
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
		vit.PostProfile(prn, "c.sys.InitChildWorkspace", body)
		ws := vit.WaitForWorkspace(wsName, prn)

		require.Empty(ws.WSError)
		require.Equal(wsName, ws.Name)
		require.Equal(it.QNameApp1_TestWSKind, ws.Kind)
		require.Equal(`{"IntFld": 10}`, ws.InitDataJSON)
		require.Equal("test_template", ws.TemplateName)
		require.Equal(istructs.ClusterID(1), ws.WSID.ClusterID())

		t.Run("check the initialized workspace using collection", func(t *testing.T) {
			body = `{"args":{"Schema":"app1pkg.air_table_plan"},"elements":[{"fields":["sys.ID","image","preview"]}]}`
			resp := vit.PostWS(ws, "q.sys.Collection", body)
			require.Len(resp.Sections[0].Elements, 2) // from testTemplate
			appEPs := vit.VVM.AppsExtensionPoints[istructs.AppQName_test1_app1]
			checkDemoAndDemoMinBLOBs(vit, "test_template", appEPs, it.QNameApp1_TestWSKind, resp, ws.WSID, prn.Token)
		})

		var idOfCDocWSKind int64

		t.Run("check current cdoc.sys.$wsKind", func(t *testing.T) {
			cdoc, id := vit.GetCDocWSKind(ws)
			idOfCDocWSKind = id
			require.Equal(float64(10), cdoc["IntFld"])
			require.Equal("", cdoc["StrFld"])
			require.Len(cdoc, 2)
		})

		t.Run("reconfigure the workspace", func(t *testing.T) {
			// CDoc<app1.WSKind> is a singleton
			body = fmt.Sprintf(`
				{
					"cuds": [
						{
							"sys.ID": %d,
							"fields": {
								"sys.QName": "app1pkg.test_ws",
								"IntFld": 42,
								"StrFld": "str"
							}
						}
					]
				}`, idOfCDocWSKind)
			vit.PostWS(ws, "c.sys.CUD", body)

			// check updated workspace config
			cdoc, _ := vit.GetCDocWSKind(ws)
			require.Len(cdoc, 2)
			require.Equal(float64(42), cdoc["IntFld"])
			require.Equal("str", cdoc["StrFld"])
		})
	})

	t.Run("create a new workspace with an existing name -> 409 conflict", func(t *testing.T) {
		body := fmt.Sprintf(`{"args": {"WSName": %q,"WSKind": "app1pkg.test_ws","WSKindInitializationData": "{\"WorkStartTime\": \"10\"}","TemplateName": "test","WSClusterID": 1}}`, wsName)
		resp := vit.PostProfile(prn, "c.sys.InitChildWorkspace", body, coreutils.Expect409())
		resp.Println()
	})

	t.Run("read user workspaces list", func(t *testing.T) {
		body := `{"args":{"Schema":"sys.ChildWorkspace"},"elements":[{"fields":["WSName","WSKind","WSID","WSError"]}]}`
		resp := vit.PostProfile(prn, "q.sys.Collection", body)
		resp.Println()
	})

	t.Run("400 bad request on create a workspace with kind that is not a QName of a workspace descriptor", func(t *testing.T) {
		wsName := vit.NextName()
		body := fmt.Sprintf(`{"args": {"WSName": %q,"WSKind": "app1pkg.articles","WSKindInitializationData": "{\"WorkStartTime\": \"10\"}","TemplateName": "test","WSClusterID": 1}}`, wsName)
		resp := vit.PostProfile(prn, "c.sys.InitChildWorkspace", body, coreutils.Expect400())
		resp.Println()
	})
}

func TestCurrentClusterIDOnMissingWSClusterID(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	wsName := vit.NextName()
	body := fmt.Sprintf(`{"args":{"WSName":%q,"WSKind":"app1pkg.test_ws","WSKindInitializationData":"{\"IntFld\": 10}"}}`, wsName)
	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
	vit.PostProfile(prn, "c.sys.InitChildWorkspace", body)
	ws := vit.WaitForWorkspace(wsName, prn)
	require.Equal(istructs.ClusterID(1), ws.WSID.ClusterID())
}

func TestWorkspaceAuthorization(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	prn := ws.Owner

	body := `{"cuds": [{"sys.ID": 1,"fields": {"sys.QName": "app1pkg.test_ws"}}]}`

	t.Run("403 forbidden", func(t *testing.T) {
		t.Run("workspace is not initialized", func(t *testing.T) {
			// try to exec c.sys.CUD in non-inited ws id 1
			vit.PostApp(istructs.AppQName_test1_app1, 1, "c.sys.CUD", body, coreutils.WithAuthorizeBy(prn.Token), coreutils.Expect403()).Println()
		})

		t.Run("access denied (wrong wsid)", func(t *testing.T) {
			// create a new login
			login := vit.SignUp(vit.NextName(), "1", istructs.AppQName_test1_app1)
			newPrn := vit.SignIn(login)

			// try to modify the workspace by the non-owner
			vit.PostApp(istructs.AppQName_test1_app1, ws.WSID, "c.sys.CUD", body, coreutils.WithAuthorizeBy(newPrn.Token), coreutils.Expect403()).Println()
		})
	})

	t.Run("401 unauthorized", func(t *testing.T) {
		t.Run("token from an another app", func(t *testing.T) {
			login := vit.SignUp(vit.NextName(), "1", istructs.AppQName_test1_app2)
			newPrn := vit.SignIn(login)
			vit.PostApp(istructs.AppQName_test1_app1, ws.WSID, "c.sys.CUD", body, coreutils.WithAuthorizeBy(newPrn.Token), coreutils.Expect401()).Println()
		})
	})
}

func TestDenyCreateCDocWSKind(t *testing.T) {
	DenyCreateCDocWSKind_Test(t, []appdef.QName{
		authnz.QNameCDoc_WorkspaceKind_UserProfile,
		authnz.QNameCDoc_WorkspaceKind_DeviceProfile,
		authnz.QNameCDoc_WorkspaceKind_AppWorkspace,
	})
}

func TestDenyCUDCDocOwnerModification(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("CDoc<ChildWorkspace>", func(t *testing.T) {
		// try to modify CDoc<ChildWorkspace>
		_, idOfCDocWSKind := vit.GetCDocChildWorkspace(ws)
		body := fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":{"WSName":"new name"}}]}`, idOfCDocWSKind) // intFld is declared in vit.SharedConfig_Simple
		vit.PostProfile(ws.Owner, "c.sys.CUD", body, coreutils.Expect403()).Println()
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

	epWSTemplates := extensionpoints.NewRootExtensionPoint()
	epTestWSKindTemplates := epWSTemplates.ExtensionPoint(it.QNameApp1_TestWSKind)
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
			_, _, err := workspace.ValidateTemplate("test"+str, epTestWSKindTemplates, it.QNameApp1_TestWSKind)
			require.Error(t, err)
			log.Println(err)
		})
	}

	t.Run("no template for workspace kind", func(t *testing.T) {
		_, _, err := workspace.ValidateTemplate("test", epTestWSKindTemplates, appdef.NewQName("sys", "unknownKind"))
		require.Error(t, err)
		log.Println(err)
	})
}

func checkDemoAndDemoMinBLOBs(vit *it.VIT, templateName string, ep extensionpoints.IExtensionPoint, wsKind appdef.QName,
	resp *coreutils.FuncResponse, wsid istructs.WSID, token string) {
	require := require.New(vit.T)
	blobs, _, err := workspace.ValidateTemplate(templateName, ep, wsKind)
	require.NoError(err)
	require.Len(blobs, 4)
	blobsMap := map[string]workspace.BLOBWorkspaceTemplateField{}
	for _, templateBLOB := range blobs {
		blobsMap[string(templateBLOB.Content)] = templateBLOB
	}
	rowIdx := 0
	for _, temp := range blobs {
		switch temp.RecordID {
		// IDs are taken from the actual templates
		case 1:
			rowIdx = 0
		case 2:
			rowIdx = 1
		default:
			vit.T.Fatal(temp.RecordID)
		}
		var fieldIdx int
		if temp.FieldName == "image" {
			fieldIdx = 1
		} else {
			fieldIdx = 2
		}
		blobID := istructs.RecordID(resp.SectionRow(rowIdx)[fieldIdx].(float64))
		uploadedBLOB := vit.GetBLOB(istructs.AppQName_test1_app1, wsid, blobID, token)
		templateBLOB := blobsMap[string(uploadedBLOB.Content)]
		require.Equal(templateBLOB.Name, uploadedBLOB.Name)
		require.Equal(templateBLOB.MimeType, uploadedBLOB.MimeType)
		delete(blobsMap, string(uploadedBLOB.Content))
		rowIdx++
	}
	require.Empty(blobsMap)
}

func TestWSNameEscaping(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")

	// \f;jf;GJ specified in frontend -> "\\f;jf;GJ" in json
	body := `{"args":{"WSName":"\\f;jf;GJ","WSKind":"app1pkg.test_ws","WSKindInitializationData":"{\"StrFld\":\"\\\\f;jf;GJ\",\"IntFld\": 10}","WSClusterID":1}}`
	vit.PostProfile(prn, "c.sys.InitChildWorkspace", body)

	vit.WaitForWorkspace(`\f;jf;GJ`, prn)
}

func TestWorkspaceInitError(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")

	wsName := vit.NextName()
	body := fmt.Sprintf(`{"args":{"WSName":"%s","WSKind":"app1pkg.test_ws","WSKindInitializationData":"{ wrong json }","WSClusterID":1}}`, wsName)
	vit.PostProfile(prn, "c.sys.InitChildWorkspace", body)

	vit.WaitForWorkspace(wsName, prn, "failed to unmarshal workspace initialization data")
}

func TestCreateChildOfChildWorkspace(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	newWSName := vit.NextName()

	body := fmt.Sprintf(`{"args": {"WSName": %q,"WSKind": "app1pkg.test_ws","WSKindInitializationData": "{\"IntFld\": 10}",
		"TemplateName": "test_template","WSClusterID": 1}}`, newWSName)
	vit.PostWS(ws, "c.sys.InitChildWorkspace", body)
	newWS := vit.WaitForChildWorkspace(ws, newWSName)

	// execute a simple operation in a new child of child
	body = `{"cuds": [{"fields": {"sys.ID": 1,"sys.QName": "app1pkg.Config","Fld1": "42"}}]}`
	// owner of ws is not owner of child of ws so let's generate a token with a WorkspaceOwner role
	// note: WSID the role belongs to must be ws, not newWS because iauthnz implementation compares wsid to ownerWSID
	pp := payloads.PrincipalPayload{
		Login:       ws.Owner.Name,
		SubjectKind: istructs.SubjectKind_User,
		ProfileWSID: ws.Owner.ProfileWSID,
		Roles:       []payloads.RoleType{{WSID: ws.WSID, QName: iauthnz.QNameRoleWorkspaceOwner}},
	}
	tokenForChildOfChild, err := vit.IssueToken(istructs.AppQName_test1_app1, time.Minute, &pp)
	require.NoError(t, err)
	vit.PostWS(newWS, "c.sys.CUD", body, coreutils.WithAuthorizeBy(tokenForChildOfChild))
}
