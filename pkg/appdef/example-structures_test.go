/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIAppDef_Structures() {

	var app appdef.IAppDef
	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	// how to build AppDef with structures
	{
		appDef := appdef.New()

		doc := appDef.AddWDoc(docName)
		doc.SetComment("This is example doc")
		doc.
			AddField("f1", appdef.DataKind_int64, true, "Field may have comments too").
			AddField("f2", appdef.DataKind_string, false)
		rec := appDef.AddWRecord(recName)

		doc.AddContainer("rec", recName, 0, appdef.Occurs_Unbounded)

		rec.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect builded AppDef with structures
	{
		cnt := 0
		app.Structures(func(s appdef.IStructure) {
			cnt++
			fmt.Printf("%d. %q: %v\n", cnt, s.QName(), s.Kind())
			fmt.Printf("- user/overall field count: %d/%d\n", s.UserFieldCount(), s.FieldCount())
			fmt.Printf("- container count: %d\n", s.ContainerCount())
		})

		fmt.Printf("Overall %d structures\n", cnt)
	}

	// Output:
	// 1. "test.doc": TypeKind_WDoc
	// - user/overall field count: 2/5
	// - container count: 1
	// 2. "test.rec": TypeKind_WRecord
	// - user/overall field count: 2/7
	// - container count: 0
	// Overall 2 structures
}
