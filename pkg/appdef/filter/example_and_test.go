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

func ExampleAnd() {
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

	fmt.Println("This example demonstrate how to work with filter And")

	example := func(flt appdef.IFilter) {
		fmt.Println("Testing", flt, "in", ws)

		for t := range ws.Types {
			fmt.Println("-", t, "is matched:", flt.Match(t))
		}

		fmt.Println("List of all matched types from", ws, ":", flt.Matches(ws))
		fmt.Println()
	}

	example(filter.And(filter.Types(appdef.TypeKind_ODoc), filter.QNames(doc)))
	example(filter.And(filter.QNames(appdef.NewQName("test", "other")), filter.Types(appdef.TypeKind_Command)))

	// Output:
	// This example demonstrate how to work with filter And
	// Testing filter And(filter Types [ODoc], filter QNames [test.doc]) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: true
	// - Object «test.object» is matched: false
	// List of all matched types from Workspace «test.workspace» : [ODoc «test.doc»]
	//
	// Testing filter And(filter QNames [test.other], filter Types [Command]) in Workspace «test.workspace»
	// - BuiltIn-Command «test.command» is matched: false
	// - ODoc «test.doc» is matched: false
	// - Object «test.object» is matched: false
	// List of all matched types from Workspace «test.workspace» : []
	//
}
