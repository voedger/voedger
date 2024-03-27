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

	// how to build AppDef with command
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		cmd := adb.AddCommand(cmdName)
		cmd.SetEngine(appdef.ExtensionEngineKind_WASM)
		cmd.
			SetParam(parName).
			SetResult(resName)
		cmd.SetUnloggedParam(unlName)

		_ = adb.AddObject(parName)
		_ = adb.AddObject(unlName)
		_ = adb.AddObject(resName)

		if a, err := adb.Build(); err == nil {
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
	}

	// Output:
	// WASM-Command «test.cmd» :
	//  - parameter: Object «test.param»
	//  - unl.param: Object «test.secure»
	//  - result   : Object «test.res»
}
