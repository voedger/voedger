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
	fmt.Println()

	wsName := appdef.NewQName("test", "workspace")
	doc, obj, cmd := appdef.NewQName("test", "doc"), appdef.NewQName("test", "object"), appdef.NewQName("test", "command")
	tag := appdef.NewQName("test", "tag")

	app := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddTag(tag)
		_ = wsb.AddODoc(doc)
		wsb.AddObject(obj).SetTag(tag)
		_ = wsb.AddCommand(cmd)

		return adb.MustBuild()
	}()

	ws := app.Workspace(wsName)

	example := func(flt appdef.IFilter) {
		fmt.Println(flt)
		fmt.Println("- kind:", flt.Kind())
		fmt.Println("- type kinds:")
		for t := range flt.Types() {
			fmt.Println("  *", t)
		}
		fmt.Println("- testing:")
		for t := range ws.LocalTypes() {
			fmt.Println("  *", t, "is matched:", flt.Match(t))
		}
		fmt.Println()
	}

	example(filter.Types(wsName, appdef.TypeKind_ODoc, appdef.TypeKind_Object))
	example(filter.Types(wsName, appdef.TypeKind_Query))

	// Output:
	// This example demonstrates how to work with the Types filter
	//
	// TYPES(ODoc, Object) FROM test.workspace
	// - kind: FilterKind_Types
	// - type kinds:
	//   * TypeKind_ODoc
	//   * TypeKind_Object
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: false
	//   * ODoc «test.doc» is matched: true
	//   * Object «test.object» is matched: true
	//   * Tag «test.tag» is matched: false
	//
	// TYPES(Query) FROM test.workspace
	// - kind: FilterKind_Types
	// - type kinds:
	//   * TypeKind_Query
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: false
	//   * ODoc «test.doc» is matched: false
	//   * Object «test.object» is matched: false
	//   * Tag «test.tag» is matched: false
}

func ExampleAllTables() {
	fmt.Println("This example demonstrates how to work with the AllTables filter")
	fmt.Println()

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
	flt := filter.AllTables(wsName)
	fmt.Println(flt)
	fmt.Println("- kind:", flt.Kind())
	fmt.Println("- testing:")
	for t := range ws.LocalTypes() {
		fmt.Println("  *", t, "is matched:", flt.Match(t))
	}

	// Output:
	// This example demonstrates how to work with the AllTables filter
	//
	// ALL TABLES FROM test.workspace
	// - kind: FilterKind_Types
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: false
	//   * ODoc «test.doc» is matched: true
	//   * Object «test.object» is matched: true
}

func ExampleAllFunctions() {
	fmt.Println("This example demonstrates how to work with the AllFunctions filter")
	fmt.Println()

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
	flt := filter.AllFunctions(wsName)
	fmt.Println(flt)
	fmt.Println("- kind:", flt.Kind())
	fmt.Println("- testing:")
	for t := range ws.LocalTypes() {
		fmt.Println("  *", t, "is matched:", flt.Match(t))
	}

	// Output:
	// This example demonstrates how to work with the AllFunctions filter
	//
	// ALL FUNCTIONS FROM test.workspace
	// - kind: FilterKind_Types
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: true
	//   * ODoc «test.doc» is matched: false
	//   * BuiltIn-Query «test.query» is matched: true
}
