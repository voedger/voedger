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

func Example() {
	fmt.Println("This example demonstrates how to work with filters")
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
		fmt.Println("- testing:")
		for _, t := range ws.LocalTypes() {
			fmt.Println("  *", t, "is matched:", flt.Match(t))
		}
		fmt.Println()
	}

	example(
		filter.And(
			filter.QNames(doc, obj),
			filter.Or(
				filter.WSTypes(wsName, appdef.TypeKind_Command),
				filter.Tags(tag))))

	example(
		filter.Or(
			filter.QNames(doc),
			filter.And(
				filter.WSTypes(wsName, appdef.TypeKind_Object),
				filter.Not(
					filter.Tags(tag)))))

	example(
		filter.Not(
			filter.Or(
				filter.QNames(doc),
				filter.Tags(tag))))

	example(filter.True())

	// Output:
	// This example demonstrates how to work with filters
	//
	// QNAMES(test.doc, test.object) AND (TYPES(Command) FROM test.workspace OR TAGS(test.tag))
	// - kind: FilterKind_And
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: false
	//   * ODoc «test.doc» is matched: false
	//   * Object «test.object» is matched: true
	//   * Tag «test.tag» is matched: false
	//
	// QNAMES(test.doc) OR (TYPES(Object) FROM test.workspace AND NOT TAGS(test.tag))
	// - kind: FilterKind_Or
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: false
	//   * ODoc «test.doc» is matched: true
	//   * Object «test.object» is matched: false
	//   * Tag «test.tag» is matched: false
	//
	// NOT (QNAMES(test.doc) OR TAGS(test.tag))
	// - kind: FilterKind_Not
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: true
	//   * ODoc «test.doc» is matched: false
	//   * Object «test.object» is matched: false
	//   * Tag «test.tag» is matched: true
	//
	// TRUE
	// - kind: FilterKind_True
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: true
	//   * ODoc «test.doc» is matched: true
	//   * Object «test.object» is matched: true
	//   * Tag «test.tag» is matched: true
}
