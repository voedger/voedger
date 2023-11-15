/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIAppDefBuilder_AddCommand() {

	var app appdef.IAppDef

	cmdName := appdef.NewQName("test", "cmd")
	parName := appdef.NewQName("test", "param")
	unlName := appdef.NewQName("test", "secure")
	resName := appdef.NewQName("test", "res")
	extName := "CmdExt"

	// how to build AppDef with command
	{
		appDef := appdef.New()

		cmd := appDef.AddCommand(cmdName)
		cmd.
			SetParam(parName).(appdef.ICommandBuilder).
			SetUnloggedParam(unlName).
			SetResult(resName).
			SetExtension(extName, appdef.ExtensionEngineKind_BuiltIn, "extension comment")

		_ = appDef.AddObject(parName)
		_ = appDef.AddObject(unlName)
		_ = appDef.AddObject(resName)

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect builded AppDef with command
	{
		cmd := app.Command(cmdName)
		fmt.Println(cmd, ":")
		fmt.Println(" - parameter:", cmd.Param())
		fmt.Println(" - unl.param:", cmd.UnloggedParam())
		fmt.Println(" - result   :", cmd.Result())
		fmt.Println(" - extension:", cmd.Extension(), cmd.Extension().Comment())
	}

	// Output:
	// Command «test.cmd» :
	//  - parameter: Object «test.param»
	//  - unl.param: Object «test.secure»
	//  - result   : Object «test.res»
	//  - extension: CmdExt (BuiltIn) extension comment
}

func ExampleIAppDef_Functions() {

	var app appdef.IAppDef

	cmdName := appdef.NewQName("test", "cmd")
	qrName := appdef.NewQName("test", "query")
	parName := appdef.NewQName("test", "param")
	resName := appdef.NewQName("test", "res")
	cmdExt := "CommandExt"
	qrExt := "QueryExt"

	// how to build AppDef with functions
	{
		appDef := appdef.New()

		appDef.AddCommand(cmdName).
			SetParam(parName).
			SetResult(resName).
			SetExtension(cmdExt, appdef.ExtensionEngineKind_WASM)

		appDef.AddQuery(qrName).
			SetParam(parName).
			SetResult(appdef.QNameANY).
			SetExtension(qrExt, appdef.ExtensionEngineKind_BuiltIn, "query extension comment")

		_ = appDef.AddObject(parName)
		_ = appDef.AddObject(resName)

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to enumerate application functions
	{
		cnt := 0
		app.Functions(func(f appdef.IFunction) {
			cnt++
			fmt.Printf("%d. %v :\n", cnt, f)
			fmt.Println(" - parameter:", f.Param())
			if c, ok := f.(appdef.ICommand); ok {
				fmt.Println(" - unl.param:", c.UnloggedParam())
			}
			fmt.Println(" - result   :", f.Result())
			fmt.Println(" - extension:", f.Extension(), f.Extension().Comment())
		})
		fmt.Printf("Overall %d function(s)", cnt)
	}

	// 1. Command «test.cmd» :
	//  - parameter: Object «test.param»
	//  - unl.param: <nil>
	//  - result   : Object «test.res»
	//  - extension: CommandExt (WASM)
	// 2. Query «test.query» :
	//  - parameter: Object «test.param»
	//  - result   : any type
	//  - extension: QueryExt (BuiltIn) query extension comment
	// Overall 2 function(s)
}
