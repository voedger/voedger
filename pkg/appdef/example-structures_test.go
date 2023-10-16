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
	objName, elName := appdef.NewQName("test", "object"), appdef.NewQName("test", "element")

	// how to build AppDef with structures
	{
		appDef := appdef.New()

		obj := appDef.AddObject(objName)
		obj.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)
		el := appDef.AddElement(elName)

		obj.AddContainer("rec", elName, 0, appdef.Occurs_Unbounded)

		el.
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
	// 1. "test.element": TypeKind_Element
	// - user/overall field count: 2/4
	// - container count: 0
	// 2. "test.object": TypeKind_Object
	// - user/overall field count: 2/3
	// - container count: 1
	// Overall 2 structures
}
