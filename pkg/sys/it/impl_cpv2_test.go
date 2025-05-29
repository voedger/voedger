/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/registry"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_CommandProcessorV2_Doc(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// insert
	body := `{
		"FldRoot": 42,
		"Nested": [
			{
				"FldNested": 43,
				"Third": [
					{"Fld1": 44},
					{"Fld1": 45}
				]
			},
			{
				"FldNested": 46,
				"Third": [
					{"Fld1": 47},
					{"Fld1": 48}
				]
			}
		]
	}`

	resp := vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Root", ws.WSID), body,
		coreutils.WithMethod(http.MethodPost),
		coreutils.WithAuthorizeBy(ws.Owner.Token),
	)
	resp.Println()
	newIDsAfterInsert := newIDs(t, resp)
	require.Equal(t, http.StatusCreated, resp.HTTPResp.StatusCode)

	path := fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Root/%d?include=Nested,Nested.Third`, ws.WSID, newIDsAfterInsert["1"])
	resp = vit.POST(path, "", coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.WithMethod(http.MethodGet))
	expectedCDoc := rootCDoc(t, newIDsAfterInsert)
	requireEqual(t, expectedCDoc, resp.Body)

	// update
	body = `{"Fld1": 100}`
	resp = vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Third/%d", ws.WSID, newIDsAfterInsert["7"]), body,
		coreutils.WithMethod(http.MethodPatch),
		coreutils.WithAuthorizeBy(ws.Owner.Token),
	)
	require.Equal(t, http.StatusOK, resp.HTTPResp.StatusCode)

	path = fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Root/%d?include=Nested,Nested.Third`, ws.WSID, newIDsAfterInsert["1"])
	resp = vit.POST(path, "", coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.WithMethod(http.MethodGet))
	resp.PrintJSON()

	expected := rootCDoc(t, newIDsAfterInsert)
	rootNestedThird := expected["Nested"].([]interface{})[1].(map[string]interface{})["Third"].([]interface{})[1].(map[string]interface{})
	rootNestedThird["Fld1"] = 100

	requireEqual(t, expected, resp.Body)

	// delete
	vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Third/%d", ws.WSID, newIDsAfterInsert["6"]), "{}",
		coreutils.WithMethod(http.MethodDelete),
		coreutils.WithAuthorizeBy(ws.Owner.Token),
	)

	path = fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Root/%d?include=Nested,Nested.Third`, ws.WSID, newIDsAfterInsert["1"])
	resp = vit.POST(path, "", coreutils.WithAuthorizeBy(ws.Owner.Token), coreutils.WithMethod(http.MethodGet))
	resp.PrintJSON()

	rootNestedThird = expected["Nested"].([]interface{})[1].(map[string]interface{})["Third"].([]interface{})[0].(map[string]interface{})
	rootNestedThird[appdef.SystemField_IsActive] = false

	requireEqual(t, expected, resp.Body)

	t.Run("insert with explicit sys.ID", func(t *testing.T) {
		body := `{
			"FldRoot": 42,
			"Nested": [
				{
					"FldNested": 43,
					"Third": [
						{"Fld1": 44},
						{"Fld1": 45}
					]
				},
				{
					"FldNested": 46,
					"Third": [
						{"Fld1": 47, "sys.ID": 123},
						{"Fld1": 48}
					]
				}
			]
		}`
		resp := vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Root", ws.WSID), body,
			coreutils.WithMethod(http.MethodPost),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
		)
		newIDs := newIDs(t, resp)
		require.Contains(t, newIDs, "123")
	})
}

func newIDs(t *testing.T, resp *coreutils.HTTPResponse) map[string]istructs.RecordID {
	respMap := map[string]interface{}{}
	require.NoError(t, json.Unmarshal([]byte(resp.Body), &respMap))
	res := map[string]istructs.RecordID{}
	for rawIDStr, storageIDfloat64 := range respMap["newIDs"].(map[string]interface{}) {
		res[rawIDStr] = istructs.RecordID(storageIDfloat64.(float64))
	}
	return res
}

func TestBasicUsage_CommandProcessorV2_ExecCmd(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("basic usage", func(t *testing.T) {
		body := `{"args":{"Arg1": 1}}`
		resp := vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/commands/app1pkg.TestCmd", ws.WSID), body,
			coreutils.WithMethod(http.MethodPost),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
		)
		resp.Println()
		m := map[string]interface{}{}
		require.NoError(t, json.Unmarshal([]byte(resp.Body), &m))
		result, err := json.Marshal(m["result"])
		require.NoError(t, err)
		require.JSONEq(t, `{"Int":42,"Str":"Str","sys.QName":"app1pkg.TestCmdResult"}`, string(result))
	})

	t.Run("404 not found on an unknown cmd", func(t *testing.T) {
		body := `{"args":{"Arg1": 1}}`
		vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/commands/app1pkg.Unknown", ws.WSID), body,
			coreutils.WithMethod(http.MethodPost),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect404(),
		).Println()
	})
}

func TestAuth(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("insert", func(t *testing.T) {
		body := `{"Fld1":42}`
		vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.TestDeniedCDoc", ws.WSID), body,
			coreutils.WithMethod(http.MethodPost),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect403("OperationKind_Insert app1pkg.TestDeniedCDoc: operation forbidden"),
		).Println()
	})

	t.Run("update", func(t *testing.T) {
		body := `{"Fld1":42}`
		sysPrn := vit.GetSystemPrincipal(istructs.AppQName_test1_app1)
		resp := vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.TestDeniedCDoc", ws.WSID), body,
			coreutils.WithMethod(http.MethodPost),
			coreutils.WithAuthorizeBy(sysPrn.Token),
		)
		newIDs := newIDs(t, resp)

		body = `{"Fld1":43}`
		vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.TestDeniedCDoc/%d", ws.WSID, newIDs["1"]), body,
			coreutils.WithMethod(http.MethodPatch),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect403("OperationKind_Update app1pkg.TestDeniedCDoc: operation forbidden"),
		).Println()
	})

	t.Run("delete", func(t *testing.T) {
		body := `{"Fld1":42}`
		resp := vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.DocDeactivateDenied", ws.WSID), body,
			coreutils.WithMethod(http.MethodPost),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
		)
		newIDs := newIDs(t, resp)

		vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.DocDeactivateDenied/%d", ws.WSID, newIDs["1"]), "{}",
			coreutils.WithMethod(http.MethodDelete),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect403("OperationKind_Deactivate app1pkg.DocDeactivateDenied: operation forbidden"),
		).Println()
	})

}

func TestErrorsCPv2(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	body := `{
		"FldRoot": 42,
		"Nested": [
			{
				"FldNested": 43,
				"Third": [
					{"Fld1": 44},
					{"Fld1": 45}
				]
			},
			{
				"FldNested": 46,
				"Third": [
					{"Fld1": 47},
					{"Fld1": 48}
				]
			}
		]
	}`

	resp := vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Root", ws.WSID), body,
		coreutils.WithMethod(http.MethodPost),
		coreutils.WithAuthorizeBy(ws.Owner.Token),
	)
	resp.Println()
	newIDs := newIDs(t, resp)

	t.Run("sys.ID among fields is not allowed on update", func(t *testing.T) {
		body = fmt.Sprintf(`{"Fld1": 100, "sys.ID": %d}`, newIDs["7"])
		vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Root/%d", ws.WSID, newIDs["7"]), body,
			coreutils.WithMethod(http.MethodPatch),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect400("sys.ID field is not allowed among fields to update"),
		).Println()
	})

	t.Run("record does not exist on update", func(t *testing.T) {
		body = `{"Fld1": 100}`
		t.Run("zero", func(t *testing.T) {
			vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Root/0", ws.WSID), body,
				coreutils.WithMethod(http.MethodPatch),
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect404(),
			).Println()
		})
		t.Run("non-zero", func(t *testing.T) {
			vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Root/%d", ws.WSID, istructs.NonExistingRecordID), body,
				coreutils.WithMethod(http.MethodPatch),
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect404(),
			).Println()
		})
	})

	t.Run("wrong explicit sys.ID type", func(t *testing.T) {
		body := `{"FldRoot": 42,"sys.ID": "wrong"}`
		vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Root", ws.WSID), body,
			coreutils.WithMethod(http.MethodPost),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect400(`field "sys.ID" must be json.Number`),
		).Println()
	})

	t.Run("body not allowed on delete", func(t *testing.T) {
		body := `{"FldRoot": 42}`
		vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.Root/%d", ws.WSID, newIDs["1"]), body,
			coreutils.WithMethod(http.MethodDelete),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect400("unexpected body is provided on delete"),
		).Println()
	})

	t.Run("RecordID and DocQName mismatch", func(t *testing.T) {
		t.Run("update", func(t *testing.T) {
			body := `{"FldRoot": 100000}`
			vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.category/%d", ws.WSID, newIDs["1"]), body,
				coreutils.WithMethod(http.MethodPatch),
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect400(fmt.Sprintf("record id %d leads to app1pkg.Root QName whereas app1pkg.category QName is mentioned in the request", newIDs["1"])),
			).Println()

		})

		t.Run("delete", func(t *testing.T) {
			vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.category/%d", ws.WSID, newIDs["1"]), "{}",
				coreutils.WithMethod(http.MethodDelete),
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect400(fmt.Sprintf("record id %d leads to app1pkg.Root QName whereas app1pkg.category QName is mentioned in the request", newIDs["1"])),
			).Println()
		})
	})

	t.Run("405 method not allowed on ODoc/ORecord in url on insert/update", func(t *testing.T) {
		t.Run("insert ODoc", func(t *testing.T) {
			vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.odoc1", ws.WSID), "{}",
				coreutils.WithMethod(http.MethodPost),
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect405("cannot operate on the ODoc\\Record in any way other than through command arguments"),
			).Println()
		})
		t.Run("insert ORecord", func(t *testing.T) {
			vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.orecord1", ws.WSID), "{}",
				coreutils.WithMethod(http.MethodPost),
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect405("cannot operate on the ODoc\\Record in any way other than through command arguments"),
			).Println()
		})

		t.Run("ODoc", func(t *testing.T) {
			body := `{"args":{"sys.ID": 1}}`
			resp := vit.PostWS(ws, "c.app1pkg.CmdODocOne", body)
			odocID := resp.NewID()
			t.Run("update", func(t *testing.T) {
				vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.odoc1/%d", ws.WSID, odocID), "{}",
					coreutils.WithMethod(http.MethodPatch),
					coreutils.WithAuthorizeBy(ws.Owner.Token),
					coreutils.Expect405("cannot operate on the ODoc\\Record in any way other than through command arguments"),
				).Println()
			})
			t.Run("delete", func(t *testing.T) {
				vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.odoc1/%d", ws.WSID, odocID), "{}",
					coreutils.WithMethod(http.MethodDelete),
					coreutils.WithAuthorizeBy(ws.Owner.Token),
					coreutils.Expect405("cannot operate on the ODoc\\Record in any way other than through command arguments"),
				).Println()
			})
		})

		t.Run("ORecord", func(t *testing.T) {
			body := `{"args":{"sys.ID": 1,"orecord1":[{"sys.ID":2,"sys.ParentID":1}]}}`
			resp := vit.PostWS(ws, "c.app1pkg.CmdODocOne", body)
			orecordID := resp.NewIDs["2"]
			t.Run("update", func(t *testing.T) {
				vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.orecord1/%d", ws.WSID, orecordID), "{}",
					coreutils.WithMethod(http.MethodPatch),
					coreutils.WithAuthorizeBy(ws.Owner.Token),
					coreutils.Expect405("cannot operate on the ODoc\\Record in any way other than through command arguments"),
				).Println()
			})
			t.Run("delete", func(t *testing.T) {
				vit.POST(fmt.Sprintf("api/v2/apps/test1/app1/workspaces/%d/docs/app1pkg.orecord1/%d", ws.WSID, orecordID), "{}",
					coreutils.WithMethod(http.MethodDelete),
					coreutils.WithAuthorizeBy(ws.Owner.Token),
					coreutils.Expect405("cannot operate on the ODoc\\Record in any way other than through command arguments"),
				).Println()
			})
		})
	})
}

// [~server.users/it.TestUsersCreate~impl]
func TestUsersCreate(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	login := vit.NextName() + "@123.com"
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, login, istructs.CurrentClusterID())
	appWSID := coreutils.GetAppWSID(pseudoWSID, istructs.DefaultNumAppWorkspaces)
	p := payloads.VerifiedValuePayload{
		VerificationKind: appdef.VerificationKind_EMail,
		WSID:             appWSID,
		Field:            "Email", // CreateEmailLoginParams.Email
		Value:            login,
		Entity:           appdef.NewQName(registry.RegistryPackage, "CreateEmailLoginParams"),
	}
	verifiedEmailToken, err := vit.ITokens.IssueToken(istructs.AppQName_sys_registry, 10*time.Minute, &p)
	require.NoError(err)
	body := fmt.Sprintf(`{"verifiedEmailToken": "%s","password": "123","displayName": "%s"}`, verifiedEmailToken, login)
	vit.POST("api/v2/apps/test1/app1/users", body).Println()

	// try to sign in
	prn := vit.SignIn(it.Login{Name: login, Pwd: "123", AppQName: istructs.AppQName_test1_app1})
	log.Println(prn)
}

func rootCDoc(t *testing.T, newIDs map[string]istructs.RecordID) map[string]interface{} {
	docJSON := fmt.Sprintf(`
		{
			"FldRoot": 42,
			"Nested": [
				{
					"FldNested": 43,
					"Third": [
						{
							"Fld1": 44,
							"sys.Container": "Third",
							"sys.ID": %[3]d,
							"sys.IsActive": true,
							"sys.ParentID": %[2]d,
							"sys.QName": "app1pkg.Third"
						},
						{
							"Fld1": 45,
							"sys.Container": "Third",
							"sys.ID": %[4]d,
							"sys.IsActive": true,
							"sys.ParentID": %[2]d,
							"sys.QName": "app1pkg.Third"
						}
					],
					"sys.Container": "Nested",
					"sys.ID": %[2]d,
					"sys.IsActive": true,
					"sys.ParentID": %[1]d,
					"sys.QName": "app1pkg.Nested"
				},
				{
					"FldNested": 46,
					"Third": [
						{
							"Fld1": 47,
							"sys.Container": "Third",
							"sys.ID": %[6]d,
							"sys.IsActive": true,
							"sys.ParentID": %[5]d,
							"sys.QName": "app1pkg.Third"
						},
						{
							"Fld1": 48,
							"sys.Container": "Third",
							"sys.ID": %[7]d,
							"sys.IsActive": true,
							"sys.ParentID": %[5]d,
							"sys.QName": "app1pkg.Third"
						}
					],
					"sys.Container": "Nested",
					"sys.ID": %[5]d,
					"sys.IsActive": true,
					"sys.ParentID": %[1]d,
					"sys.QName": "app1pkg.Nested"
				}
			],
			"sys.ID": %[1]d,
			"sys.IsActive": true,
			"sys.QName": "app1pkg.Root"
		}`, newIDs["1"], newIDs["2"], newIDs["3"], newIDs["4"], newIDs["5"], newIDs["6"], newIDs["7"])
	res := map[string]interface{}{}
	require.NoError(t, json.Unmarshal([]byte(docJSON), &res))
	return res
}

func requireEqual(t *testing.T, expected map[string]interface{}, actualJSON string) {
	expectedJSON, err := json.Marshal(&expected)
	require.NoError(t, err)
	require.JSONEq(t, string(expectedJSON), actualJSON)
}
