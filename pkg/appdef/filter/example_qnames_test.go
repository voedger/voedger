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

func ExampleQNames() {
	wsName := appdef.NewQName("test", "workspace")
	doc1, doc2, doc3 := appdef.NewQName("test", "doc1"), appdef.NewQName("test", "doc2"), appdef.NewQName("test", "doc3")

	app := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddODoc(doc1)
		_ = wsb.AddODoc(doc2)
		_ = wsb.AddODoc(doc3)

		return adb.MustBuild()
	}()

	flt := filter.QNames(doc1, doc2)

	fmt.Println("This example demonstrate how to work with filter", flt.Kind().TrimString())

	ws := app.Workspace(wsName)

	fmt.Println("Testing", flt, "in", ws)

	for doc := range appdef.ODocs(ws) {
		fmt.Println("-", doc, "is matched:", flt.Match(doc))
	}

	fmt.Println("List of all matched types from", ws, ":", flt.Matches(ws))

	// Output:
	// This example demonstrate how to work with filter QNames
	// Testing filter QNames [test.doc1 test.doc2] in Workspace «test.workspace»
	// - ODoc «test.doc1» is matched: true
	// - ODoc «test.doc2» is matched: true
	// - ODoc «test.doc3» is matched: false
	// List of all matched types from Workspace «test.workspace» : [ODoc «test.doc1», ODoc «test.doc2»]
}
