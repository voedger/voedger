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

func Example() {
	fmt.Println("This example demonstrates how to work with filters")
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
		fmt.Println("- testing:")
		for t := range ws.LocalTypes() {
			fmt.Println("  *", t, "is matched:", flt.Match(t))
		}
		fmt.Println()
	}

	example(
		filter.And(
			filter.QNames(doc, obj),
			filter.Or(
				filter.Types(wsName, appdef.TypeKind_Command),
				filter.Tags(tag))))

	example(
		filter.Or(
			filter.QNames(doc),
			filter.And(
				filter.Types(wsName, appdef.TypeKind_Object),
				filter.Tags(tag))))

	example(
		filter.Not(
			filter.Or(
				filter.QNames(doc),
				filter.Tags(tag))))

	// Output:
	// This example demonstrates how to work with filters
	//
	// QNames(test.doc, test.object) and (Types(Command) from Workspace test.workspace or Tags(test.tag))
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: false
	//   * ODoc «test.doc» is matched: false
	//   * Object «test.object» is matched: true
	//   * Tag «test.tag» is matched: false
	//
	// QNames(test.doc) or (Types(Object) from Workspace test.workspace and Tags(test.tag))
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: false
	//   * ODoc «test.doc» is matched: true
	//   * Object «test.object» is matched: true
	//   * Tag «test.tag» is matched: false
	//
	// not (QNames(test.doc) or Tags(test.tag))
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: true
	//   * ODoc «test.doc» is matched: false
	//   * Object «test.object» is matched: false
	//   * Tag «test.tag» is matched: true
}
