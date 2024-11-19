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

func ExampleTypes() {
	fmt.Println("This example demonstrates how to work with the Types filter")

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
		fmt.Println("The", flt, "Types:")
		for t := range flt.Types() {
			fmt.Println("-", t)
		}

		fmt.Println("Testing", flt, "in", ws)
		for t := range ws.LocalTypes {
			fmt.Println("-", t, "is matched:", flt.Match(t))
		}
	}

	example(filter.Types(appdef.TypeKind_ODoc, appdef.TypeKind_Object))
	example(filter.Types(appdef.TypeKind_Query))

	// Output:
	// This example demonstrates how to work with the Types filter
	//
	// The filter.Types(ODoc, Object) Types:
	// - TypeKind_ODoc
	// - TypeKind_Object
	// Testing filter.Types(ODoc, Object) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: true
	// - Object «test.object» is matched: true
	//
	// The filter.Types(Query) Types:
	// - TypeKind_Query
	// Testing filter.Types(Query) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: false
	// - Object «test.object» is matched: false
}

func ExampleAllTables() {
	fmt.Println("This example demonstrates how to work with the AllTables filter")

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
	flt := filter.AllTables()
	fmt.Println("Testing filter AllTables in", ws)
	for t := range ws.LocalTypes {
		fmt.Println("-", t, "is matched:", flt.Match(t))
	}

	// Output:
	// This example demonstrates how to work with the AllTables filter
	// Testing filter AllTables in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: true
	// - Object «test.object» is matched: true
}

func ExampleAllFunctions() {
	fmt.Println("This example demonstrates how to work with the AllFunctions filter")

	wsName := appdef.NewQName("test", "workspace")
	doc, cmd, qry := appdef.NewQName("test", "doc"), appdef.NewQName("test", "command"), appdef.NewQName("test", "query")

	app := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddODoc(doc)
		_ = wsb.AddCommand(cmd)
		_ = wsb.AddQuery(qry)

		return adb.MustBuild()
	}()

	ws := app.Workspace(wsName)
	flt := filter.AllFunctions()
	fmt.Println("Testing filter AllFunctions in", ws)
	for t := range ws.LocalTypes {
		fmt.Println("-", t, "is matched:", flt.Match(t))
	}

	// Output:
	// This example demonstrates how to work with the AllFunctions filter
	// Testing filter AllFunctions in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: true
	// - ODoc «test.doc» is matched: false
	// - BuiltIn-Query «test.query» is matched: true
}
