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

func ExampleNot() {
	fmt.Println("This example demonstrates how to work with the Not filter")

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
		fmt.Println("The filter", flt, "negative sub-filter:")
		fmt.Println("-", flt.Not())
		fmt.Println("Testing filter", flt, "in", ws)
		for t := range ws.LocalTypes() {
			fmt.Println("-", t, "is matched:", flt.Match(t))
		}
	}

	example(filter.Not(filter.Types(wsName, appdef.TypeKind_Command)))
	example(filter.Not(filter.Or(filter.QNames(doc), filter.QNames(obj))))

	// Output:
	// This example demonstrates how to work with the Not filter
	//
	// The filter not Types(Command from Workspace test.workspace) negative sub-filter:
	// - Types(Command from Workspace test.workspace)
	// Testing filter not Types(Command from Workspace test.workspace) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: true
	// - Object «test.object» is matched: true
	//
	// The filter not (QNames(test.doc) or QNames(test.object)) negative sub-filter:
	// - QNames(test.doc) or QNames(test.object)
	// Testing filter not (QNames(test.doc) or QNames(test.object)) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: true
	// - ODoc «test.doc» is matched: false
	// - Object «test.object» is matched: false
}
