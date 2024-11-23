/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
)

func Test_View(t *testing.T) {

	var app appdef.IAppDef
	viewName := appdef.NewQName("test", "view")

	// prepare AppDef with view
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		docName := appdef.NewQName("test", "doc")
		_ = wsb.AddCDoc(docName)

		view := wsb.AddView(viewName)
		view.SetComment("view comment")
		view.Key().PartKey().
			AddField("pk_int", appdef.DataKind_int64).
			AddRefField("pk_ref", docName)
		view.Key().ClustCols().
			AddField("cc_int", appdef.DataKind_int64).
			AddRefField("cc_ref", docName).
			AddField("cc_name", appdef.DataKind_string, appdef.MaxLen(100))
		view.Value().
			AddField("vv_int", appdef.DataKind_int64, true).
			AddRefField("vv_ref", true, docName).
			AddField("vv_code", appdef.DataKind_string, false, appdef.MaxLen(10), appdef.Pattern(`^\w+$`)).
			AddField("vv_data", appdef.DataKind_bytes, false, appdef.MaxLen(1024)).SetFieldComment("vv_data", "One kilobyte of data")
		if a, err := adb.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	{
		v := newView()
		v.read(appdef.View(app.Type, viewName))

		json, err := json.Marshal(v)

		require := require.New(t)
		require.NoError(err)

		// os.WriteFile("C://temp//view_test.json", json, 0644)

		const expected = `{
			"Comment": "view comment",
			"Key": {
				"Partition": [
					{
						"Name": "pk_int",
						"Data": "sys.int64",
						"Required": true
					},
					{
						"Name": "pk_ref",
						"Data": "sys.RecordID",
						"Required": true,
						"Refs": [
							"test.doc"
						]
					}
				],
				"ClustCols": [
					{
						"Name": "cc_int",
						"Data": "sys.int64"
					},
					{
						"Name": "cc_ref",
						"Data": "sys.RecordID",
						"Refs": [
							"test.doc"
						]
					},
					{
						"Name": "cc_name",
						"DataType": {
							"Ancestor": "sys.string",
							"Constraints": {
								"MaxLen": 100
							}
						}
					}
				]
			},
			"Value": [
				{
					"Name": "sys.QName",
					"Data": "sys.QName",
					"Required": true
				},
				{
					"Name": "vv_int",
					"Data": "sys.int64",
					"Required": true
				},
				{
					"Name": "vv_ref",
					"Data": "sys.RecordID",
					"Required": true,
					"Refs": [
						"test.doc"
					]
				},
				{
					"Name": "vv_code",
					"DataType": {
						"Ancestor": "sys.string",
						"Constraints": {
							"MaxLen": 10,
							"Pattern": "^\\w+$"
						}
					}
				},
				{
					"Comment": "One kilobyte of data",
					"Name": "vv_data",
					"DataType": {
						"Ancestor": "sys.bytes",
						"Constraints": {
							"MaxLen": 1024
						}
					}
				}
			]
		}`
		require.JSONEq(expected, string(json))
	}
}
