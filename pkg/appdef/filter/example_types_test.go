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

	flt := filter.Types(appdef.TypeKind_ODoc, appdef.TypeKind_Object)

	fmt.Println("This example demonstrate how to work with filter", flt.Kind().TrimString())

	ws := app.Workspace(wsName)

	fmt.Println("test", flt, "in", ws)

	for t := range ws.Types {
		fmt.Println(t, "is matched:", flt.Match(t))
	}

	fmt.Println("list of all matched types from", ws, ":", flt.Matches(ws))

	// Output:
	// This example demonstrate how to work with filter Types
	// test filter Types [ODoc Object] in Workspace «test.workspace»
	// BuiltIn-Command «test.command» is matched: false
	// ODoc «test.doc» is matched: true
	// Object «test.object» is matched: true
	// list of all matched types from Workspace «test.workspace» : [test.doc test.object]
}
