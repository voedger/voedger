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
	argName := appdef.NewQName("test", "arg")
	unlName := appdef.NewQName("test", "unl")
	resName := appdef.NewQName("test", "res")
	extName := "CmdExt"

	// how to build AppDef with command
	{
		appDef := appdef.New()

		cmd := appDef.AddCommand(cmdName)
		cmd.
			SetArg(argName).(appdef.ICommandBuilder).
			SetUnloggedArg(unlName).
			SetResult(resName).
			SetExtension(extName, appdef.ExtensionEngineKind_BuiltIn)

		_ = appDef.AddObject(argName)
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
		fmt.Println(" - argument :", cmd.Arg())
		fmt.Println(" - unl.arg. :", cmd.UnloggedArg())
		fmt.Println(" - result   :", cmd.Result())
		fmt.Println(" - extension:", cmd.Extension())
	}

	// Output:
	// Command «test.cmd» :
	//  - argument : Object «test.arg»
	//  - unl.arg. : Object «test.unl»
	//  - result   : Object «test.res»
	//  - extension: CmdExt (BuiltIn)
}
