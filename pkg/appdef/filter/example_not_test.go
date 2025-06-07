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

func ExampleNot() {
	fmt.Println("This example demonstrates how to work with the Not filter")
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
		fmt.Println("- not:", flt.Not())
		fmt.Println("- testing:")
		for _, t := range ws.LocalTypes() {
			fmt.Println("  *", t, "is matched:", flt.Match(t))
		}
		fmt.Println()
	}

	example(filter.Not(filter.WSTypes(wsName, appdef.TypeKind_Command)))
	example(filter.Not(filter.Or(filter.QNames(doc), filter.QNames(obj))))

	// Output:
	// This example demonstrates how to work with the Not filter
	//
	// NOT TYPES(Command) FROM test.workspace
	// - kind: FilterKind_Not
	// - not: TYPES(Command) FROM test.workspace
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: false
	//   * ODoc «test.doc» is matched: true
	//   * Object «test.object» is matched: true
	//   * Tag «test.tag» is matched: true
	//
	// NOT (QNAMES(test.doc) OR QNAMES(test.object))
	// - kind: FilterKind_Not
	// - not: QNAMES(test.doc) OR QNAMES(test.object)
	// - testing:
	//   * BuiltIn-Command «test.command» is matched: true
	//   * ODoc «test.doc» is matched: false
	//   * Object «test.object» is matched: false
	//   * Tag «test.tag» is matched: true
}
