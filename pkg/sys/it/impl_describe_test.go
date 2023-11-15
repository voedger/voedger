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
		require.Contains(names, "app1pkg")
	})

	t.Run("describe package", func(t *testing.T) {
		body := `{"args":{"PackageName":"my"},"elements":[{"fields":["PackageDesc"]}]}`
		desc := vit.PostProfile(prn, "q.sys.DescribePackage", body).SectionRow()[0].(string)

		actual := map[string]interface{}{}
		require.NoError(json.Unmarshal([]byte(desc), &actual))

		expected := map[string]interface{}{
			"Name": "my",
			"Views": map[string]interface{}{
				"app1.View": map[string]interface{}{
					"Name": "my.View",
					"Key": map[string]interface{}{
						"ClustCols": []interface{}{
							map[string]interface{}{
								"Name": "ViewStrFld",
								"Data": "sys.string"}},
						"Partition": []interface{}{
							map[string]interface{}{
								"Name":     "ViewIntFld",
								"Data":     "sys.int32",
								"Required": true}}},
					"Value": []interface{}{
						map[string]interface{}{
							"Name":     "sys.QName",
							"Data":     "sys.QName",
							"Required": true},
						map[string]interface{}{
							"Name": "ViewByteFld",
							"DataType": map[string]interface{}{
								"Ancestor": "sys.bytes",
								"Constraints": map[string]interface{}{
									"MaxLen": 512.0}}}}}}}
		require.EqualValues(expected, actual)
	})
}
