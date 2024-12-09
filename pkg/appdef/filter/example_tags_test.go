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

func ExampleTags() {
	fmt.Println("This example demonstrates how to work with the Tags filter")

	wsName := appdef.NewQName("test", "workspace")
	doc1, doc2, doc3 := appdef.NewQName("test", "doc1"), appdef.NewQName("test", "doc2"), appdef.NewQName("test", "doc3")
	tagOdd := appdef.NewQName("test", "tagOdd")
	tagEven := appdef.NewQName("test", "tagEven")

	app := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddTag(tagOdd)
		wsb.AddTag(tagEven)

		wsb.AddODoc(doc1).SetTag(tagOdd)
		wsb.AddODoc(doc2).SetTag(tagEven)
		wsb.AddODoc(doc3).SetTag(tagOdd)

		return adb.MustBuild()
	}()

	ws := app.Workspace(wsName)

	example := func(flt appdef.IFilter) {
		fmt.Println()
		fmt.Println("The filter", flt, "Tags:")
		for t := range flt.Tags() {
			fmt.Println("-", t)
		}

		fmt.Println("Testing filter", flt, "in", ws)
		for t := range appdef.ODocs(ws.LocalTypes()) {
			fmt.Println("-", t, "is matched:", flt.Match(t))
		}
	}

	example(filter.Tags(tagOdd))
	example(filter.Tags(tagEven))
	example(filter.Tags(tagOdd, tagEven))

	// Output:
	// This example demonstrates how to work with the Tags filter
	//
	// The filter Tags(test.tagOdd) Tags:
	// - test.tagOdd
	// Testing filter Tags(test.tagOdd) in Workspace «test.workspace»
	// - ODoc «test.doc1» is matched: true
	// - ODoc «test.doc2» is matched: false
	// - ODoc «test.doc3» is matched: true
	//
	// The filter Tags(test.tagEven) Tags:
	// - test.tagEven
	// Testing filter Tags(test.tagEven) in Workspace «test.workspace»
	// - ODoc «test.doc1» is matched: false
	// - ODoc «test.doc2» is matched: true
	// - ODoc «test.doc3» is matched: false
	//
	// The filter Tags(test.tagEven, test.tagOdd) Tags:
	// - test.tagEven
	// - test.tagOdd
	// Testing filter Tags(test.tagEven, test.tagOdd) in Workspace «test.workspace»
	// - ODoc «test.doc1» is matched: true
	// - ODoc «test.doc2» is matched: true
	// - ODoc «test.doc3» is matched: true
}
