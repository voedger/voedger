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

func ExampleQueries() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")

	qryName := appdef.NewQName("test", "query")
	parName := appdef.NewQName("test", "param")
	resName := appdef.NewQName("test", "res")

	// how to build AppDef with query
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		qry := wsb.AddQuery(qryName)
		qry.SetEngine(appdef.ExtensionEngineKind_WASM)
		qry.
			SetParam(parName).
			SetResult(resName)

		_ = wsb.AddObject(parName)
		_ = wsb.AddObject(resName)

		app = adb.MustBuild()
	}

	// how to enum queries
	{
		cnt := 0
		for q := range appdef.Queries(app.Types()) {
			cnt++
			fmt.Println(cnt, q)
		}
		fmt.Println("overall:", cnt)
	}

	// how to inspect builded AppDef with query
	{
		qry := appdef.Query(app.Type, qryName)
		fmt.Println(qry, ":")
		fmt.Println(" - parameter:", qry.Param())
		fmt.Println(" - result   :", qry.Result())
	}

	// Output:
	// 1 WASM-Query «test.query»
	// overall: 1
	// WASM-Query «test.query» :
	//  - parameter: Object «test.param»
	//  - result   : Object «test.res»
}
