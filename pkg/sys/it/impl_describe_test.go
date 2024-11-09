/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_DescribeSchema(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	prnApp1 := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")
	prnApp2 := vit.GetPrincipal(istructs.AppQName_test1_app2, "login")

	t.Run("describe package names", func(t *testing.T) {
		body := `{"args":{},"elements":[{"fields":["Names"]}]}`
		namesStr := vit.PostProfile(prnApp1, "q.sys.DescribePackageNames", body).SectionRow()[0].(string)
		names := strings.Split(namesStr, ",")
		require.Len(names, 1)
		require.Contains(names, "app1pkg")
	})

	t.Run("describe package", func(t *testing.T) {
		body := `{"args":{"PackageName":"app2pkg"},"elements":[{"fields":["PackageDesc"]}]}`
		desc := vit.PostProfile(prnApp2, "q.sys.DescribePackage", body).SectionRow()[0].(string)

		actual := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(desc), &actual))

		t.Run("check workspace description", func(t *testing.T) {
			require.Len(actual["Workspaces"], 1)
			ws := actual["Workspaces"].(map[string]interface{})["app2pkg.test_wsWS"].(map[string]interface{})

			require.Equal("app2pkg.test_ws", ws["Descriptor"])

			t.Run("check workspace structures", func(t *testing.T) {
				structs := ws["Structures"].(map[string]interface{})
				require.Len(structs, 2)

				t.Run("check app2pkg.test_ws", func(t *testing.T) {
					require.Contains(structs, "app2pkg.test_ws")
					require.EqualValues(
						map[string]interface{}{
							"Fields": []interface{}{
								map[string]interface{}{
									"Data":     "sys.QName",
									"Name":     "sys.QName",
									"Required": true,
								}, map[string]interface{}{
									"Data":     "sys.RecordID",
									"Name":     "sys.ID",
									"Required": true,
								}, map[string]interface{}{
									"Data": "sys.bool",
									"Name": "sys.IsActive",
								}, map[string]interface{}{
									"Data":     "sys.int32",
									"Name":     "IntFld",
									"Required": true,
								}, map[string]interface{}{
									"Data": "sys.string",
									"Name": "StrFld",
								},
							},
							"Kind":      "CDoc",
							"Singleton": true,
						},
						structs["app2pkg.test_ws"].(map[string]interface{}),
					)
				})

				t.Run("check app2pkg.doc1", func(t *testing.T) {
					require.Contains(structs, "app2pkg.doc1")
					require.EqualValues(
						map[string]interface{}{
							"Fields": []interface{}{
								map[string]interface{}{
									"Data":     "sys.QName",
									"Name":     "sys.QName",
									"Required": true,
								}, map[string]interface{}{
									"Data":     "sys.RecordID",
									"Name":     "sys.ID",
									"Required": true,
								}, map[string]interface{}{
									"Data": "sys.bool",
									"Name": "sys.IsActive",
								},
							},
							"Kind": "CDoc",
						},
						structs["app2pkg.doc1"].(map[string]interface{}),
					)
				})
			})

			t.Run("check workspace extensions", func(t *testing.T) {
				extensions := ws["Extensions"].(map[string]interface{})
				require.Len(extensions, 1)
				commands := extensions["Commands"].(map[string]interface{})
				require.Len(commands, 1)
				t.Run("check app2pkg.testCmd", func(t *testing.T) {
					require.Contains(commands, "app2pkg.testCmd")
					require.EqualValues(
						map[string]interface{}{
							"Engine": "BuiltIn",
							"Name":   "testCmd",
						},
						commands["app2pkg.testCmd"].(map[string]interface{}),
					)
				})
			})
		})
	})
}
