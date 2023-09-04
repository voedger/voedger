/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIView() {

	var app appdef.IAppDef
	viewName := appdef.NewQName("test", "view")

	// how to build AppDef with view
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
			AddBytesField("vv_data", false, appdef.MaxLen(1024))
		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect view
	{
		// how to find view by name
		view := app.View(viewName)
		fmt.Printf("view %q: %v, %s\n", view.QName(), view.Kind(), view.Comment())

		field := func(f appdef.IField) {
			fmt.Printf("- %s: %s", f.Name(), f.DataKind().TrimString())
			if f.IsSys() {
				fmt.Print(", sys")
			}
			if f.Required() {
				fmt.Print(", required")
			}
			if r, ok := f.(appdef.IRefField); ok {
				if len(r.Refs()) != 0 {
					fmt.Printf(", refs: %v", r.Refs())
				}
			}
			if s, ok := f.(appdef.IStringField); ok {
				fmt.Printf(", restricts: %v", s.Restricts())
			}
			fmt.Println()
		}

		// how to inspect view partition key
		fmt.Printf("view partition key is %q, %d fields:\n", view.Key().Partition().QName(), view.Key().Partition().FieldCount())
		view.Key().Partition().Fields(field)

		// how to inspect view clustering columns
		fmt.Printf("view clustering columns key is %q, %d fields:\n", view.Key().ClustCols().QName(), view.Key().ClustCols().FieldCount())
		view.Key().ClustCols().Fields(field)

		// how to inspect view value
		fmt.Printf("view value is %q, %d fields:\n", view.Value().QName(), view.Value().FieldCount())
		view.Value().Fields(field)
	}

	// Output:
	// view "test.view": DefKind_ViewRecord, view comment
	// view partition key is "test.view_PartitionKey", 2 fields:
	// - pk_int: int64, required
	// - pk_ref: RecordID, required, refs: [test.doc]
	// view clustering columns key is "test.view_ClusteringColumns", 3 fields:
	// - cc_int: int64
	// - cc_ref: RecordID, refs: [test.doc]
	// - cc_name: string, restricts: MaxLen: 100
	// view value is "test.view_Value", 5 fields:
	// - sys.QName: QName, sys, required
	// - vv_int: int64, required
	// - vv_ref: RecordID, required, refs: [test.doc]
	// - vv_code: string, restricts: MaxLen: 10, Pattern: `^\w+$`
	// - vv_data: bytes, restricts: MaxLen: 1024
}
