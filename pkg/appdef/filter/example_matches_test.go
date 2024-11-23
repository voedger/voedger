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

func ExampleMatches() {
	fmt.Println("This example demonstrates how to use Matches() func")

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

	flt := filter.And(filter.Types(appdef.TypeKind_ODoc), filter.QNames(doc))

	fmt.Println("Matches for", flt, "in", ws, ":")
	for t := range filter.Matches(flt, ws.LocalTypes) {
		fmt.Println("-", t)
	}

	// Output:
	// This example demonstrates how to use Matches() func
	// Matches for filter.And(filter.Types(ODoc), filter.QNames(test.doc)) in Workspace «test.workspace» :
	// - ODoc «test.doc»
}
