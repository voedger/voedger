/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
)

func ExampleTypes() {
	fmt.Println("This example demonstrates how to work with the Types filter")
	fmt.Println()

	wsName := appdef.NewQName("test", "workspace")
	doc, obj, cmd := appdef.NewQName("test", "doc"), appdef.NewQName("test", "object"), appdef.NewQName("test", "command")
	tag := appdef.NewQName("test", "tag")

	app := func() appdef.IAppDef {
		adb := builder.New()
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
		for _, t := range flt.Types() {
			fmt.Println("  *", t)
		}
		fmt.Println("- testing:")
		for _, t := range ws.LocalTypes() {
			fmt.Println("  *", t, "is matched:", flt.Match(t))
		}
		fmt.Println()
	}

	example(filter.Types(appdef.TypeKind_ODoc, appdef.TypeKind_Object))
	example(filter.Types(appdef.TypeKind_Query))

	// Output:
	// This example demonstrates how to work with the Types filter
	//
	// TYPES(ODoc, Object)
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
	// TYPES(Query)
	// - kind: FilterKind_Types
	// - type kinds:
	//   * TypeKind_Query
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: false
	//   * ODoc «test.doc» is matched: false
	//   * Object «test.object» is matched: false
	//   * Tag «test.tag» is matched: false
}

func ExampleWSTypes() {
	fmt.Println("This example demonstrates how to work with the WSTypes filter")
	fmt.Println()

	wsName := appdef.NewQName("test", "workspace")
	doc, obj, cmd := appdef.NewQName("test", "doc"), appdef.NewQName("test", "object"), appdef.NewQName("test", "command")
	tag := appdef.NewQName("test", "tag")

	app := func() appdef.IAppDef {
		adb := builder.New()
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
		for _, t := range flt.Types() {
			fmt.Println("  *", t)
		}
		fmt.Println("- testing:")
		for _, t := range ws.LocalTypes() {
			fmt.Println("  *", t, "is matched:", flt.Match(t))
		}
		fmt.Println()
	}

	example(filter.WSTypes(wsName, appdef.TypeKind_ODoc, appdef.TypeKind_Object))
	example(filter.WSTypes(wsName, appdef.TypeKind_Query))

	// Output:
	// This example demonstrates how to work with the WSTypes filter
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

	ws1 := appdef.NewQName("test", "workspace1")
	doc1, obj1, cmd1 := appdef.NewQName("test", "doc1"), appdef.NewQName("test", "object1"), appdef.NewQName("test", "command1")
	ws2 := appdef.NewQName("test", "workspace2")
	doc2, obj2, cmd2 := appdef.NewQName("test", "doc2"), appdef.NewQName("test", "object2"), appdef.NewQName("test", "command2")

	app := func() appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb1 := adb.AddWorkspace(ws1)
		wsb1.SetAbstract()
		_ = wsb1.AddODoc(doc1)
		_ = wsb1.AddObject(obj1)
		_ = wsb1.AddCommand(cmd1)

		wsb2 := adb.AddWorkspace(ws2)
		wsb2.SetAncestors(ws1)
		_ = wsb2.AddODoc(doc2)
		_ = wsb2.AddObject(obj2)
		_ = wsb2.AddCommand(cmd2)

		return adb.MustBuild()
	}()

	flt := filter.AllTables()
	fmt.Println(flt)
	fmt.Println("- kind:", flt.Kind())
	fmt.Println("- testing:")
	for _, t := range app.Types() {
		if !t.IsSystem() {
			fmt.Println("  *", t, "is matched:", flt.Match(t))
		}
	}

	// Output:
	// This example demonstrates how to work with the AllTables filter
	//
	// ALL TABLES
	// - kind: FilterKind_Types
	// - testing:
	//   * BuiltIn-Command «test.command1» is matched: false
	//   * BuiltIn-Command «test.command2» is matched: false
	//   * ODoc «test.doc1» is matched: true
	//   * ODoc «test.doc2» is matched: true
	//   * Object «test.object1» is matched: true
	//   * Object «test.object2» is matched: true
	//   * Workspace «test.workspace1» is matched: false
	//   * Workspace «test.workspace2» is matched: false
}

func ExampleAllWSTables() {
	fmt.Println("This example demonstrates how to work with the AllWSTables filter")
	fmt.Println()

	wsName := appdef.NewQName("test", "workspace")
	doc, obj, cmd := appdef.NewQName("test", "doc"), appdef.NewQName("test", "object"), appdef.NewQName("test", "command")

	app := func() appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddODoc(doc)
		_ = wsb.AddObject(obj)
		_ = wsb.AddCommand(cmd)

		return adb.MustBuild()
	}()

	ws := app.Workspace(wsName)
	flt := filter.AllWSTables(wsName)
	fmt.Println(flt)
	fmt.Println("- kind:", flt.Kind())
	fmt.Println("- testing:")
	for _, t := range ws.LocalTypes() {
		fmt.Println("  *", t, "is matched:", flt.Match(t))
	}

	// Output:
	// This example demonstrates how to work with the AllWSTables filter
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

	ws1 := appdef.NewQName("test", "workspace1")
	doc1, cmd1, qry1 := appdef.NewQName("test", "doc1"), appdef.NewQName("test", "command1"), appdef.NewQName("test", "query1")
	ws2 := appdef.NewQName("test", "workspace2")
	doc2, cmd2, qry2 := appdef.NewQName("test", "doc2"), appdef.NewQName("test", "command2"), appdef.NewQName("test", "query2")

	app := func() appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb1 := adb.AddWorkspace(ws1)
		wsb1.SetAbstract()
		_ = wsb1.AddODoc(doc1)
		_ = wsb1.AddCommand(cmd1)
		_ = wsb1.AddQuery(qry1)

		wsb2 := adb.AddWorkspace(ws2)
		wsb2.SetAncestors(ws1)
		_ = wsb2.AddODoc(doc2)
		_ = wsb2.AddCommand(cmd2)
		_ = wsb2.AddQuery(qry2)

		return adb.MustBuild()
	}()

	flt := filter.AllFunctions()
	fmt.Println(flt)
	fmt.Println("- kind:", flt.Kind())
	fmt.Println("- testing:")
	for _, t := range app.Types() {
		if !t.IsSystem() {
			fmt.Println("  *", t, "is matched:", flt.Match(t))
		}
	}

	// Output:
	// This example demonstrates how to work with the AllFunctions filter
	//
	// ALL FUNCTIONS
	// - kind: FilterKind_Types
	// - testing:
	//   * BuiltIn-Command «test.command1» is matched: true
	//   * BuiltIn-Command «test.command2» is matched: true
	//   * ODoc «test.doc1» is matched: false
	//   * ODoc «test.doc2» is matched: false
	//   * BuiltIn-Query «test.query1» is matched: true
	//   * BuiltIn-Query «test.query2» is matched: true
	//   * Workspace «test.workspace1» is matched: false
	//   * Workspace «test.workspace2» is matched: false
}

func ExampleAllWSFunctions() {
	fmt.Println("This example demonstrates how to work with the AllFunctions filter")
	fmt.Println()

	wsName := appdef.NewQName("test", "workspace")
	doc, cmd, qry := appdef.NewQName("test", "doc"), appdef.NewQName("test", "command"), appdef.NewQName("test", "query")

	app := func() appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddODoc(doc)
		_ = wsb.AddCommand(cmd)
		_ = wsb.AddQuery(qry)

		return adb.MustBuild()
	}()

	ws := app.Workspace(wsName)
	flt := filter.AllWSFunctions(wsName)
	fmt.Println(flt)
	fmt.Println("- kind:", flt.Kind())
	fmt.Println("- testing:")
	for _, t := range ws.LocalTypes() {
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
