/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
)

func ExampleFunctions() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")

	cmdName := appdef.NewQName("test", "cmd")
	parName := appdef.NewQName("test", "param")
	resName := appdef.NewQName("test", "res")
	queryName := appdef.NewQName("test", "query")

	// how to build AppDef with functions
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		cmd := wsb.AddCommand(cmdName)
		cmd.SetEngine(appdef.ExtensionEngineKind_WASM)
		cmd.
			SetParam(parName).
			SetResult(resName)

		query := wsb.AddQuery(queryName)
		query.SetResult(resName)

		_ = wsb.AddObject(parName)
		_ = wsb.AddObject(resName)

		app = adb.MustBuild()
	}

	// how to enum functions
	{
		cnt := 0
		for f := range appdef.Functions(app.Types()) {
			cnt++
			fmt.Println(cnt, f)
		}
		fmt.Println("overall function(s):", cnt)
	}

	// how to find functions
	{
		cmd := appdef.Function(app.Type, cmdName)
		fmt.Println(cmd, ":")
		fmt.Println(" - parameter:", cmd.Param())
		fmt.Println(" - result   :", cmd.Result())

		query := appdef.Function(app.Type, queryName)
		fmt.Println(query, ":")
		fmt.Println(" - parameter:", query.Param())
		fmt.Println(" - result   :", query.Result())

		fmt.Println("Search unknown:", appdef.Function(app.Type, appdef.NewQName("test", "unknown")))
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
