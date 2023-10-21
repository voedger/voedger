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
		appDef := appdef.New()

		docName := appdef.NewQName("test", "doc")
		_ = appDef.AddCDoc(docName)

		view := appDef.AddView(viewName)
		view.SetComment("view comment")
		view.KeyBuilder().PartKeyBuilder().
			AddField("pk_int", appdef.DataKind_int64).
			AddRefField("pk_ref", docName)
		view.KeyBuilder().ClustColsBuilder().
			AddField("cc_int", appdef.DataKind_int64).
			AddRefField("cc_ref", docName).
			AddStringField("cc_name", 100)
		view.ValueBuilder().
			AddField("vv_int", appdef.DataKind_int64, true).
			AddRefField("vv_ref", true, docName).
			AddStringField("vv_code", false, appdef.MaxLen(10), appdef.Pattern(`^\w+$`)).
			AddBytesField("vv_data", false, appdef.MaxLen(1024)).
			SetFieldComment("vv_data", "One kilobyte of data")
		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	{
		v := newView()
		v.read(app.View(viewName))

		json, err := json.Marshal(v)

		require := require.New(t)
		require.NoError(err)

		//ioutil.WriteFile("C://temp//view_test.json", json, 0644)

		const expected = `{
			"Comment": "view comment",
			"Name": "test.view",
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
