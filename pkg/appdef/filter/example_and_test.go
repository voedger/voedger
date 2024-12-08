/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
)

func ExampleAnd() {
	fmt.Println("This example demonstrates how to work with the And filter")

	wsName := appdef.NewQName("test", "workspace")
	doc, obj, cmd := appdef.NewQName("test", "doc"), appdef.NewQName("test", "object"), appdef.NewQName("test", "command")

	app := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddODoc(doc)
		_ = wsb.AddObject(obj)
		_ = wsb.AddCommand(cmd)

		return adb.MustBuild()
	}()

	ws := app.Workspace(wsName)

	example := func(flt appdef.IFilter) {
		fmt.Println()
		fmt.Println("The filter", flt, "children:")
		for f := range flt.And() {
			fmt.Println("-", f)
		}
		fmt.Println("Testing filter", flt, "in", ws)
		for t := range ws.LocalTypes() {
			fmt.Println("-", t, "is matched:", flt.Match(t))
		}
	}

	example(filter.And(filter.Types(wsName, appdef.TypeKind_ODoc), filter.QNames(doc)))
	example(filter.And(filter.QNames(appdef.NewQName("test", "other")), filter.Types(wsName, appdef.TypeKind_Command)))

	// Output:
	// This example demonstrates how to work with the And filter
	//
	// The filter Types(ODoc from Workspace test.workspace) and QNames(test.doc) children:
	// - Types(ODoc from Workspace test.workspace)
	// - QNames(test.doc)
	// Testing filter Types(ODoc from Workspace test.workspace) and QNames(test.doc) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: true
	// - Object «test.object» is matched: false
	//
	// The filter QNames(test.other) and Types(Command from Workspace test.workspace) children:
	// - QNames(test.other)
	// - Types(Command from Workspace test.workspace)
	// Testing filter QNames(test.other) and Types(Command from Workspace test.workspace) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: false
	// - Object «test.object» is matched: false
}
