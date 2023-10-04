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
		view.Key().Partition().
			AddField("pk_int", appdef.DataKind_int64).
			AddRefField("pk_ref", docName)
		view.Key().ClustCols().
			AddField("cc_int", appdef.DataKind_int64).
			AddRefField("cc_ref", docName).
			AddStringField("cc_name", 100)
		view.Value().
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

		const expected = `{
	"Comment": "view comment", 
	"Key": {
		"ClustCols": [
			{
				"Kind":	"DataKind_int64", 
				"Name":"cc_int"
			}, 
			{
				"Kind": "DataKind_RecordID", 
				"Name": "cc_ref", 
				"Refs": ["test.doc"]
			}, 
			{
				"Kind": "DataKind_string", 
				"Name": "cc_name", 
				"Restricts": {
					"MaxLen":	100
				}
			}
		], 
		"Partition":[
			{
				"Kind":	"DataKind_int64", 
				"Name":	"pk_int", 
				"Required":	true
			}, 
			{
				"Kind":	"DataKind_RecordID", 
				"Name":	"pk_ref", 
				"Refs": ["test.doc"], 
				"Required":	true
			}
		]
	}, 
	"Name"	:	"test.view", 
	"Value":	[
		{
			"Kind": "DataKind_QName", 
			"Name": "sys.QName", 
			"Required":	true
		}, 
		{
			"Kind": "DataKind_int64", 
			"Name": "vv_int", 
			"Required": true
		}, 
		{
			"Kind": "DataKind_RecordID", 
			"Name": "vv_ref", 
			"Refs": ["test.doc"], 
			"Required": true
		}, 
		{
			"Kind": "DataKind_string", 
			"Name": "vv_code", 
			"Restricts": {
				"MaxLen":	10, 
				"Pattern":	"^\\w+$"
			}
		}, 
		{
			"Comment": "One kilobyte of data",
			"Kind": "DataKind_bytes", 
			"Name":	"vv_data", 
			"Restricts": {
				"MaxLen":1024
			}
		}
	]
}`
		require.JSONEq(expected, string(json))
	}
}
