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
		fmt.Println("The", flt, "Not() filter:")
		fmt.Println("-", flt.Not())
		fmt.Println("Testing", flt, "in", ws)
		for t := range ws.LocalTypes {
			fmt.Println("-", t, "is matched:", flt.Match(t))
		}
	}

	example(filter.Not(filter.Types(appdef.TypeKind_Command)))
	example(filter.Not(filter.QNames(doc, obj)))

	// Output:
	// This example demonstrates how to work with the Not filter
	//
	// The filter.Not(filter.Types(Command)) Not() filter:
	// - filter.Types(Command)
	// Testing filter.Not(filter.Types(Command)) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: true
	// - Object «test.object» is matched: true
	//
	// The filter.Not(filter.QNames(test.doc, test.object)) Not() filter:
	// - filter.QNames(test.doc, test.object)
	// Testing filter.Not(filter.QNames(test.doc, test.object)) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: true
	// - ODoc «test.doc» is matched: false
	// - Object «test.object» is matched: false
}
