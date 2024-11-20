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
	fmt.Println("This example demonstrates how to work with the QNames filter")

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

	ws := app.Workspace(wsName)

	example := func(flt appdef.IFilter) {
		fmt.Println()
		fmt.Println("The", flt, "QNames:")
		for n := range flt.QNames() {
			fmt.Println("-", n)
		}

		fmt.Println("Testing", flt, "in", ws)
		for t := range ws.LocalTypes {
			fmt.Println("-", t, "is matched:", flt.Match(t))
		}
	}

	example(filter.QNames(doc1, doc2))
	example(filter.QNames(appdef.NewQName("test", "unknown")))

	// Output:
	// This example demonstrates how to work with the QNames filter
	//
	// The filter.QNames(test.doc1, test.doc2) QNames:
	// - test.doc1
	// - test.doc2
	// Testing filter.QNames(test.doc1, test.doc2) in Workspace «test.workspace»
	// - ODoc «test.doc1» is matched: true
	// - ODoc «test.doc2» is matched: true
	// - ODoc «test.doc3» is matched: false
	//
	// The filter.QNames(test.unknown) QNames:
	// - test.unknown
	// Testing filter.QNames(test.unknown) in Workspace «test.workspace»
	// - ODoc «test.doc1» is matched: false
	// - ODoc «test.doc2» is matched: false
	// - ODoc «test.doc3» is matched: false
}
