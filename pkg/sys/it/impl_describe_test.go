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
		require.Len(names, 2)
		require.Contains(names, "sys")
		require.Contains(names, "app1pkg")
	})

	t.Run("describe package", func(t *testing.T) {
		body := `{"args":{"PackageName":"app2pkg"},"elements":[{"fields":["PackageDesc"]}]}`
		desc := vit.PostProfile(prnApp2, "q.sys.DescribePackage", body).SectionRow()[0].(string)

		actual := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(desc), &actual))

		expected := map[string]interface{}{
			"Structures": map[string]interface{}{
				"app2pkg.test_ws": map[string]interface{}{
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
				"app2pkg.doc1": map[string]interface{}{
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
			},
			"Extensions": map[string]interface{}{
				"Commands": map[string]interface{}{
					"app2pkg.testCmd": map[string]interface{}{
						"Engine": "BuiltIn",
						"Name":   "testCmd",
					},
				},
			},
		}
		require.EqualValues(expected, actual)
	})
}
