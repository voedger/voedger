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

func ExampleOr() {
	fmt.Println("This example demonstrates how to work with the Or filter")
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
		fmt.Println("- children:")
		for _, f := range flt.Or() {
			fmt.Println("  *", f)
		}
		fmt.Println("- testing:")
		for _, t := range ws.LocalTypes() {
			fmt.Println("  *", t, "is matched:", flt.Match(t))
		}
		fmt.Println()
	}

	example(
		filter.Or(
			filter.WSTypes(wsName, appdef.TypeKind_ODoc),
			filter.Tags(tag)))

	example(
		filter.Or(
			filter.QNames(doc),
			filter.And(
				filter.WSTypes(wsName, appdef.TypeKind_Object),
				filter.Tags(tag))))

	// Output:
	// This example demonstrates how to work with the Or filter
	//
	// TYPES(ODoc) FROM test.workspace OR TAGS(test.tag)
	// - kind: FilterKind_Or
	// - children:
	//   * TYPES(ODoc) FROM test.workspace
	//   * TAGS(test.tag)
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: false
	//   * ODoc «test.doc» is matched: true
	//   * Object «test.object» is matched: true
	//   * Tag «test.tag» is matched: false
	//
	// QNAMES(test.doc) OR (TYPES(Object) FROM test.workspace AND TAGS(test.tag))
	// - kind: FilterKind_Or
	// - children:
	//   * QNAMES(test.doc)
	//   * TYPES(Object) FROM test.workspace AND TAGS(test.tag)
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: false
	//   * ODoc «test.doc» is matched: true
	//   * Object «test.object» is matched: true
	//   * Tag «test.tag» is matched: false
}
