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
	wsName := appdef.NewQName("test", "workspace")

	doc := appdef.NewQName("test", "doc")
	obj := appdef.NewQName("test", "object")
	cmd := appdef.NewQName("test", "command")

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
		fmt.Println("Testing", flt, "in", ws)
		for t := range ws.LocalTypes {
			fmt.Println("-", t, "is matched:", flt.Match(t))
		}

		fmt.Println("All matches:")
		cnt := 0
		for t := range filter.Matches(flt, ws.LocalTypes) {
			cnt++
			fmt.Println("-", t)
		}
		if cnt == 0 {
			fmt.Println("- no matches")
		}
	}

	example(filter.And(filter.Types(appdef.TypeKind_ODoc), filter.QNames(doc)))
	example(filter.And(filter.QNames(appdef.NewQName("test", "other")), filter.Types(appdef.TypeKind_Command)))

	// Output:
	//
	// Testing filter.And(filter.Types(ODoc), filter.QNames(test.doc)) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: true
	// - Object «test.object» is matched: false
	// All matches:
	// - ODoc «test.doc»
	//
	// Testing filter.And(filter.QNames(test.other), filter.Types(Command)) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: false
	// - Object «test.object» is matched: false
	// All matches:
	// - no matches
}
