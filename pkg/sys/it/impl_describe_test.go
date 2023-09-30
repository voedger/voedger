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
	vit := it.NewVIT(t, &it.SharedConfig_Simple)
	defer vit.TearDown()

	prn := vit.GetPrincipal(istructs.AppQName_test1_app1, "login")

	t.Run("describe package names", func(t *testing.T) {
		body := `{"args":{},"elements":[{"fields":["Names"]}]}`
		namesStr := vit.PostProfile(prn, "q.sys.DescribePackageNames", body).SectionRow()[0].(string)
		names := strings.Split(namesStr, ",")
		require.Len(names, 3)
		require.Contains(names, "sys")
		require.Contains(names, "my")
		require.Contains(names, "simpleApp")
	})

	t.Run("describe package", func(t *testing.T) {
		body := `{"args":{"PackageName":"my"},"elements":[{"fields":["PackageDesc"]}]}`
		desc := vit.PostProfile(prn, "q.sys.DescribePackage", body).SectionRow()[0].(string)

		actual := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(desc), &actual))

		expected := map[string]interface{}{
			"Name": "my",
			"Views": map[string]interface{}{
				"my.View": map[string]interface{}{
					"Key": map[string]interface{}{
						"ClustCols": []interface{}{map[string]interface{}{
							"Kind": "DataKind_string",
							"Name": "ViewStrFld"}},
						"Partition": []interface{}{map[string]interface{}{
							"Kind":     "DataKind_int32",
							"Name":     "ViewIntFld",
							"Required": true}}},
					"Name": "my.View",
					"Value": []interface{}{
						map[string]interface{}{
							"Kind":     "DataKind_QName",
							"Name":     "sys.QName",
							"Required": true},
						map[string]interface{}{
							"Kind": "DataKind_bytes",
							"Name": "ViewByteFld",
							"Restricts": map[string]interface{}{
								"MaxLen": 512.0}}}}}}
		require.Equal(expected, actual)
	})
}
