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

	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")

	t.Run("describe package names", func(t *testing.T) {
		body := `{"args":{},"elements":[{"fields":["Names"]}]}`
		namesStr := vit.PostProfile(prn, "q.sys.DescribePackageNames", body).SectionRow()[0].(string)
		names := strings.Split(namesStr, ",")
		require.Len(names, 3)
		require.Contains(names, "sys")
		require.Contains(names, "my")
		require.Contains(names, "app1")
	})

	t.Run("describe package", func(t *testing.T) {
		body := `{"args":{"PackageName":"my"},"elements":[{"fields":["PackageDesc"]}]}`
		desc := vit.PostProfile(prn, "q.sys.DescribePackage", body).SectionRow()[0].(string)

		actual := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(desc), &actual))

		expected := map[string]interface{}{
			"Name": "my",
			"Types": map[string]interface{}{
				"my.View": map[string]interface{}{
					"Containers": []interface{}{
						map[string]interface{}{"MaxOccurs": float64(1), "MinOccurs": float64(1), "Name": "sys.key", "Type": "my.View_FullKey"},
						map[string]interface{}{"MaxOccurs": float64(1), "MinOccurs": float64(1), "Name": "sys.val", "Type": "my.View_Value"},
					},
					"Kind": "TypeKind_ViewRecord",
					"Name": "my.View",
				},
				"my.View_FullKey": map[string]interface{}{
					"Fields": []interface{}{
						map[string]interface{}{"Kind": "DataKind_int32", "Name": "ViewIntFld", "Required": true},
						map[string]interface{}{"Kind": "DataKind_string", "Name": "ViewStrFld"},
					},
					"Containers": []interface{}{
						map[string]interface{}{"MaxOccurs": float64(1), "MinOccurs": float64(1), "Name": "sys.pkey", "Type": "my.View_PartitionKey"},
						map[string]interface{}{"MaxOccurs": float64(1), "MinOccurs": float64(1), "Name": "sys.ccols", "Type": "my.View_ClusteringColumns"},
					},
					"Kind": "TypeKind_ViewRecord_Key",
					"Name": "my.View_FullKey",
				},
				"my.View_PartitionKey": map[string]interface{}{
					"Fields": []interface{}{map[string]interface{}{"Kind": "DataKind_int32", "Name": "ViewIntFld", "Required": true}},
					"Kind":   "TypeKind_ViewRecord_PartitionKey",
					"Name":   "my.View_PartitionKey",
				},
				"my.View_ClusteringColumns": map[string]interface{}{
					"Fields": []interface{}{map[string]interface{}{"Kind": "DataKind_string", "Name": "ViewStrFld"}},
					"Kind":   "TypeKind_ViewRecord_ClusteringColumns",
					"Name":   "my.View_ClusteringColumns",
				},
				"my.View_Value": map[string]interface{}{
					"Fields": []interface{}{map[string]interface{}{"Kind": "DataKind_QName", "Name": "sys.QName", "Required": true}},
					"Kind":   "TypeKind_ViewRecord_Value",
					"Name":   "my.View_Value",
				},
			},
		}
		require.Equal(expected, actual)
	})
}
