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

func ExampleTags() {
	fmt.Println("This example demonstrates how to work with the Tags filter")
	fmt.Println()

	wsName := appdef.NewQName("test", "workspace")
	doc1, doc2, doc3 := appdef.NewQName("test", "doc1"), appdef.NewQName("test", "doc2"), appdef.NewQName("test", "doc3")
	tagOdd := appdef.NewQName("test", "tagOdd")
	tagEven := appdef.NewQName("test", "tagEven")

	app := func() appdef.IAppDef {
		adb := builder.New()
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
		fmt.Println(flt)
		fmt.Println("- kind:", flt.Kind())
		fmt.Println("- tags:")
		for _, t := range flt.Tags() {
			fmt.Println("  *", t)
		}
		fmt.Println("- testing:")
		for t := range appdef.ODocs(ws.LocalTypes()) {
			fmt.Println("  *", t, "is matched:", flt.Match(t))
		}
		fmt.Println()
	}

	example(filter.Tags(tagOdd))
	example(filter.Tags(tagEven))
	example(filter.Tags(tagOdd, tagEven))

	// Output:
	// This example demonstrates how to work with the Tags filter
	//
	// TAGS(test.tagOdd)
	// - kind: FilterKind_Tags
	// - tags:
	//   * test.tagOdd
	// - testing:
	//   * ODoc «test.doc1» is matched: true
	//   * ODoc «test.doc2» is matched: false
	//   * ODoc «test.doc3» is matched: true
	//
	// TAGS(test.tagEven)
	// - kind: FilterKind_Tags
	// - tags:
	//   * test.tagEven
	// - testing:
	//   * ODoc «test.doc1» is matched: false
	//   * ODoc «test.doc2» is matched: true
	//   * ODoc «test.doc3» is matched: false
	//
	// TAGS(test.tagEven, test.tagOdd)
	// - kind: FilterKind_Tags
	// - tags:
	//   * test.tagEven
	//   * test.tagOdd
	// - testing:
	//   * ODoc «test.doc1» is matched: true
	//   * ODoc «test.doc2» is matched: true
	//   * ODoc «test.doc3» is matched: true
}
