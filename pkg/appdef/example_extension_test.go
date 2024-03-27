/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIAppDef_Extensions() {

	var app appdef.IAppDef

	cmdName := appdef.NewQName("test", "cmd")
	qrName := appdef.NewQName("test", "query")
	prjName := appdef.NewQName("test", "projector")
	parName := appdef.NewQName("test", "param")
	resName := appdef.NewQName("test", "res")

	sysViews := appdef.NewQName("sys", "views")
	viewName := appdef.NewQName("test", "view")

	// how to build AppDef with extensions
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		cmd := adb.AddCommand(cmdName)
		cmd.SetEngine(appdef.ExtensionEngineKind_WASM)
		cmd.
			SetParam(parName).
			SetResult(resName)

		qry := adb.AddQuery(qrName)
		qry.
			SetParam(parName).
			SetResult(appdef.QNameANY)

		prj := adb.AddProjector(prjName)
		prj.Events().
			Add(cmdName, appdef.ProjectorEventKind_Execute)
		prj.Intents().
			Add(sysViews, viewName)

		_ = adb.AddObject(parName)
		_ = adb.AddObject(resName)

		if a, err := adb.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to enumerate extensions
	{
		cnt := 0
		app.Extensions(func(ex appdef.IExtension) {
			cnt++
			fmt.Printf("%d. %v :\n", cnt, ex)
			switch ex.Kind() {
			case appdef.TypeKind_Command:
				cmd := ex.(appdef.ICommand)
				fmt.Println(" - parameter:", cmd.Param())
				fmt.Println(" - unl.param:", cmd.UnloggedParam())
				fmt.Println(" - result   :", cmd.Result())
			case appdef.TypeKind_Query:
				qry := ex.(appdef.IQuery)
				fmt.Println(" - parameter:", qry.Param())
				fmt.Println(" - result   :", qry.Result())
			case appdef.TypeKind_Projector:
				prj := ex.(appdef.IProjector)
				prj.Events().Enum(func(e appdef.IProjectorEvent) {
					fmt.Println(" - event    :", e)
				})
				prj.States().Enum(func(s appdef.IStorage) {
					fmt.Println(" - state    :", s)
				})
				prj.Intents().Enum(func(s appdef.IStorage) {
					fmt.Println(" - intent   :", s)
				})
			}
		})
		fmt.Printf("Overall %d extension(s)", cnt)
	}

	// how to find extension
	{
		ext := app.Extension(cmdName)
		fmt.Println("Extension:", ext)
	}

	// 1. WASM-Command «test.cmd» :
	//  - parameter: Object «test.param»
	//  - unl.param: <nil>
	//  - result   : Object «test.res»
	// 2. BuiltIn-Query «test.query» :
	//  - parameter: Object «test.param»
	//  - result   : any type
	// 3. BuiltIn-Projector «test.projector» :
	//  - event    : Command «test.cmd» [Execute]
	//  - intent   : Storage «sys.views» [test.view]
	// Overall 3 extensions(s)
	// Extension: WASM-Command «test.cmd»
}
