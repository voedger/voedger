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

func ExampleOr() {
	fmt.Println("This example demonstrates how to work with the Or filter")

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
		fmt.Println("The", flt, "Or() children:")
		for f := range flt.Or() {
			fmt.Println("-", f)
		}
		fmt.Println("Testing", flt, "in", ws)
		for t := range ws.LocalTypes() {
			fmt.Println("-", t, "is matched:", flt.Match(t))
		}
	}

	example(filter.Or(filter.Types(wsName, appdef.TypeKind_ODoc), filter.QNames(obj)))
	example(filter.Or(filter.QNames(appdef.NewQName("test", "other")), filter.Types(wsName, appdef.TypeKind_Command)))

	// Output:
	// This example demonstrates how to work with the Or filter
	//
	// The filter.Or(filter.Types(workspace «test.workspace»: ODoc), filter.QNames(test.object)) Or() children:
	// - filter.Types(workspace «test.workspace»: ODoc)
	// - filter.QNames(test.object)
	// Testing filter.Or(filter.Types(workspace «test.workspace»: ODoc), filter.QNames(test.object)) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: true
	// - Object «test.object» is matched: true
	//
	// The filter.Or(filter.QNames(test.other), filter.Types(workspace «test.workspace»: Command)) Or() children:
	// - filter.QNames(test.other)
	// - filter.Types(workspace «test.workspace»: Command)
	// Testing filter.Or(filter.QNames(test.other), filter.Types(workspace «test.workspace»: Command)) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: true
	// - ODoc «test.doc» is matched: false
	// - Object «test.object» is matched: false
}
