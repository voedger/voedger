# Filter Package

The `filter` package provides a set of utilities for filtering types in the `appdef` package by various criteria such as qualified names, tags, and type kinds. It also includes logical combinations for combining filters.

Filters should be used for building projectors, ACL rules, Rate limits and other.

## General Example
  
```go
package main

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
)

func main() {
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
}
```

To run this example in The Go Playground, copy the code above and paste it into the editor at [The Go Playground](https://play.golang.org/). Then click the "Run" button to execute the code.

## Filters

### QNames

`QNames` is a filter that matches types by their qualified names.

### Tags

`Tags` is a filter that matches types by their tags.

**Note:** This filter is not ready for use yet (deprecated).

### Types

`Types` is a filter that matches types by their type kinds.

#### Special filters based on Types

`AllTables` is a filter that matches all tables.

`AllFunctions` is a filter that matches all functions.

## Logical Combinations

`And` is a logical combinator that combines multiple filters using a logical AND operation.

`Or` is a logical combinator that combines multiple filters using a logical OR operation.

`Not` is a logical combinator that negates a filter.
