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
		fmt.Printf("view partition key has %d fields:\n", view.Key().PartKey().FieldCount())
		view.Key().PartKey().Fields(field)

		// how to inspect view clustering columns
		fmt.Printf("view clustering columns key has %d fields:\n", view.Key().ClustCols().FieldCount())
		view.Key().ClustCols().Fields(field)

		// how to inspect view value
		fmt.Printf("view value has %d fields:\n", view.Value().FieldCount())
		view.Value().Fields(field)
	}

	// Output:
	// view "test.view": TypeKind_ViewRecord, view comment
	// view partition key has 2 fields:
	// - pk_int: int64, required
	// - pk_ref: RecordID, required, refs: [test.doc]
	// view clustering columns key has 3 fields:
	// - cc_int: int64
	// - cc_ref: RecordID, refs: [test.doc]
	// - cc_name: string, restricts: MaxLen: 100
	// view value has 5 fields:
	// - sys.QName: QName, sys, required
	// - vv_int: int64, required
	// - vv_ref: RecordID, required, refs: [test.doc]
	// - vv_code: string, restricts: MaxLen: 10, Pattern: `^\w+$`
	// - vv_data: bytes, restricts: MaxLen: 1024
}
