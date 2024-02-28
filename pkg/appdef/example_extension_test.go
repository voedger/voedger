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

	// how to build AppDef with extensions
	{
		appDef := appdef.New()

		cmd := appDef.AddCommand(cmdName)
		cmd.SetEngine(appdef.ExtensionEngineKind_WASM)
		cmd.
			SetParam(parName).
			SetResult(resName)

		qry := appDef.AddQuery(qrName)
		qry.
			SetParam(parName).
			SetResult(appdef.QNameANY)

		prj := appDef.AddProjector(prjName)
		prj.
			AddEvent(cmdName, appdef.ProjectorEventKind_Execute)

		_ = appDef.AddObject(parName)
		_ = appDef.AddObject(resName)

		if a, err := appDef.Build(); err == nil {
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
				prj.Events(func(e appdef.IProjectorEvent) {
					fmt.Println(" - event    :", e)
				})
				prj.States(func(s appdef.QName, names appdef.QNames) {
					fmt.Println(" - state    :", s, names)
				})
				prj.Intents(func(s appdef.QName, names appdef.QNames) {
					fmt.Println(" - intent   :", s, names)
				})
			}
		})
		fmt.Printf("Overall %d extension(s)", cnt)
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
	// Overall 3 extensions(s)
}
