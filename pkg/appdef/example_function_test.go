/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIAppDef_Functions() {

	var app appdef.IAppDef

	cmdName := appdef.NewQName("test", "cmd")
	parName := appdef.NewQName("test", "param")
	resName := appdef.NewQName("test", "res")
	queryName := appdef.NewQName("test", "query")

	// how to build AppDef with functions
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		cmd := adb.AddCommand(cmdName)
		cmd.SetEngine(appdef.ExtensionEngineKind_WASM)
		cmd.
			SetParam(parName).
			SetResult(resName)

		query := adb.AddQuery(queryName)
		query.SetResult(resName)

		_ = adb.AddObject(parName)
		_ = adb.AddObject(resName)

		app = adb.MustBuild()
	}

	// how to enum functions
	{
		cnt := 0
		for f := range app.Functions {
			cnt++
			fmt.Println(cnt, f)
		}
		fmt.Println("overall function(s):", cnt)
	}

	// how to find functions
	{
		cmd := app.Function(cmdName)
		fmt.Println(cmd, ":")
		fmt.Println(" - parameter:", cmd.Param())
		fmt.Println(" - result   :", cmd.Result())

		query := app.Function(queryName)
		fmt.Println(query, ":")
		fmt.Println(" - parameter:", query.Param())
		fmt.Println(" - result   :", query.Result())

		fmt.Println("Search unknown:", app.Function(appdef.NewQName("test", "unknown")))
	}

	// Output:
	// 1 WASM-Command «test.cmd»
	// 2 BuiltIn-Query «test.query»
	// overall function(s): 2
	// WASM-Command «test.cmd» :
	//  - parameter: Object «test.param»
	//  - result   : Object «test.res»
	// BuiltIn-Query «test.query» :
	//  - parameter: <nil>
	//  - result   : Object «test.res»
	// Search unknown: <nil>
}
