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

		view := appDef.AddView(viewName)
		view.SetComment("view comment")
		view.Key().Partition().
			AddField("pk1", appdef.DataKind_int64).
			AddField("pk2", appdef.DataKind_bool)
		view.Key().ClustCols().
			AddField("cc1", appdef.DataKind_float64).
			AddStringField("cc2", 100)
		view.Value().
			AddField("v1", appdef.DataKind_float64, true).
			AddBytesField("v2", false, 1024)

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

		// how to inspect view partition key
		fmt.Printf("view partition key is %q, %d fields:\n", view.Key().Partition().QName(), view.Key().Partition().FieldCount())
		view.Key().Partition().Fields(func(f appdef.IField) {
			fmt.Printf("- %s: %s\n", f.Name(), f.DataKind().TrimString())
		})

		// how to inspect view clustering columns
		fmt.Printf("view clustering columns key is %q, %d fields:\n", view.Key().ClustCols().QName(), view.Key().ClustCols().FieldCount())
		view.Key().ClustCols().Fields(func(f appdef.IField) {
			fmt.Printf("- %s: %s\n", f.Name(), f.DataKind().TrimString())
		})

		// how to inspect view value
		fmt.Printf("view value is %q, %d fields:\n", view.Value().QName(), view.Value().FieldCount())
		view.Value().Fields(func(f appdef.IField) {
			fmt.Printf("- %s: %s, required: %v\n", f.Name(), f.DataKind().TrimString(), f.Required())
		})
	}

	// Output:
	// view "test.view": DefKind_ViewRecord, view comment
	// view partition key is "test.view_PartitionKey", 2 fields:
	// - pk1: int64
	// - pk2: bool
	// view clustering columns key is "test.view_ClusteringColumns", 2 fields:
	// - cc1: float64
	// - cc2: string
	// view value is "test.view_Value", 3 fields:
	// - sys.QName: QName, required: true
	// - v1: float64, required: true
	// - v2: bytes, required: false
}
