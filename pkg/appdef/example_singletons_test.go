/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIAppDef_Singletons() {

	var app appdef.IAppDef
	wsName := appdef.NewQName("test", "workspace")
	cDocName := appdef.NewQName("test", "cdoc")
	wDocName := appdef.NewQName("test", "wdoc")

	// how to build AppDef with singletons
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		cDoc := wsb.AddCDoc(cDocName)
		cDoc.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)
		cDoc.SetSingleton()

		wDoc := wsb.AddWDoc(wDocName)
		wDoc.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)
		wDoc.SetSingleton()

		app = adb.MustBuild()
	}

	// how to inspect builded AppDef with singletons
	{
		cnt := 0
		for s := range app.Singletons {
			cnt++
			fmt.Printf("%d. %v\n", cnt, s)
		}

		fmt.Printf("Overall %d singletons\n", cnt)
	}

	// how to find singleton by name
	{
		fmt.Println(app.Singleton(cDocName))
		fmt.Println(app.Singleton(wDocName))
		fmt.Println(app.Singleton(appdef.NewQName("test", "unknown")))

		ws := app.Workspace(wsName)
		fmt.Println(ws)
		fmt.Println(ws.Singleton(cDocName))
		fmt.Println(ws.Singleton(wDocName))
		fmt.Println(ws.Singleton(appdef.NewQName("test", "unknown")))
	}

	// Output:
	// 1. CDoc «test.cdoc»
	// 2. WDoc «test.wdoc»
	// Overall 2 singletons
	// CDoc «test.cdoc»
	// WDoc «test.wdoc»
	// <nil>
	// Workspace «test.workspace»
	// CDoc «test.cdoc»
	// WDoc «test.wdoc»
	// <nil>
}
