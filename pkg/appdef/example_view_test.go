/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"sort"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIView() {

	var app appdef.IAppDef
	viewName := appdef.NewQName("test", "view")

	// how to build AppDef with view
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		docName := appdef.NewQName("test", "doc")
		_ = adb.AddCDoc(docName)

		view := adb.AddView(viewName)
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
			AddField("vv_data", appdef.DataKind_bytes, false, appdef.MaxLen(1024))

		app = adb.MustBuild()
	}

	// now to enum views
	{
		cnt := 0
		for v := range app.Views {
			cnt++
			fmt.Println(cnt, v)
		}
		fmt.Println("overall view(s):", cnt)
	}

	// how to inspect view
	{
		// how to find view by name
		view := app.View(viewName)
		fmt.Printf("view %q: %v, %s\n", view.QName(), view.Kind(), view.Comment())

		fields := func(ff []appdef.IField) {
			for _, f := range ff {
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
				str := []string{}
				for _, c := range f.Constraints() {
					str = append(str, fmt.Sprint(c))
				}
				if len(str) > 0 {
					sort.Strings(str)
					fmt.Printf(", constraints: [%v]", strings.Join(str, `, `))
				}
				fmt.Println()
			}
		}

		// how to inspect all view fields
		fmt.Printf("view has %d fields:\n", view.FieldCount())
		fields(view.Fields())

		// how to inspect view key fields
		fmt.Printf("view key has %d fields:\n", view.Key().FieldCount())
		fields(view.Key().Fields())

		// how to inspect view partition key
		fmt.Printf("view partition key has %d fields:\n", view.Key().PartKey().FieldCount())
		fields(view.Key().PartKey().Fields())

		// how to inspect view clustering columns
		fmt.Printf("view clustering columns key has %d fields:\n", view.Key().ClustCols().FieldCount())
		fields(view.Key().ClustCols().Fields())

		// how to inspect view value
		fmt.Printf("view value has %d fields:\n", view.Value().FieldCount())
		fields(view.Value().Fields())
	}

	// Output:
	// 1 ViewRecord «test.view»
	// overall view(s): 1
	// view "test.view": TypeKind_ViewRecord, view comment
	// view has 10 fields:
	// - sys.QName: QName, sys, required
	// - pk_int: int64, required
	// - pk_ref: RecordID, required, refs: [test.doc]
	// - cc_int: int64
	// - cc_ref: RecordID, refs: [test.doc]
	// - cc_name: string, constraints: [MaxLen: 100]
	// - vv_int: int64, required
	// - vv_ref: RecordID, required, refs: [test.doc]
	// - vv_code: string, constraints: [MaxLen: 10, Pattern: `^\w+$`]
	// - vv_data: bytes, constraints: [MaxLen: 1024]
	// view key has 5 fields:
	// - pk_int: int64, required
	// - pk_ref: RecordID, required, refs: [test.doc]
	// - cc_int: int64
	// - cc_ref: RecordID, refs: [test.doc]
	// - cc_name: string, constraints: [MaxLen: 100]
	// view partition key has 2 fields:
	// - pk_int: int64, required
	// - pk_ref: RecordID, required, refs: [test.doc]
	// view clustering columns key has 3 fields:
	// - cc_int: int64
	// - cc_ref: RecordID, refs: [test.doc]
	// - cc_name: string, constraints: [MaxLen: 100]
	// view value has 5 fields:
	// - sys.QName: QName, sys, required
	// - vv_int: int64, required
	// - vv_ref: RecordID, required, refs: [test.doc]
	// - vv_code: string, constraints: [MaxLen: 10, Pattern: `^\w+$`]
	// - vv_data: bytes, constraints: [MaxLen: 1024]
}
