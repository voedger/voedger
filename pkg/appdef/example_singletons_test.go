/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleSingletons() {

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
		for s := range appdef.Singletons(app.Types) {
			cnt++
			fmt.Printf("%d. %v\n", cnt, s)
		}

		fmt.Printf("Overall %d singletons\n", cnt)
	}

	// how to find singleton by name
	{
		fmt.Println(appdef.Singleton(app.Type, cDocName))
		fmt.Println(appdef.Singleton(app.Type, wDocName))
		fmt.Println(appdef.Singleton(app.Type, appdef.NewQName("test", "unknown")))
	}

	// Output:
	// 1. CDoc «test.cdoc»
	// 2. WDoc «test.wdoc»
	// Overall 2 singletons
	// CDoc «test.cdoc»
	// WDoc «test.wdoc»
	// <nil>
}
