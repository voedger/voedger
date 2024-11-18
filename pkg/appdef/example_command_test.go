/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleCommands() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")

	cmdName := appdef.NewQName("test", "cmd")
	parName := appdef.NewQName("test", "param")
	unlName := appdef.NewQName("test", "secure")
	resName := appdef.NewQName("test", "res")

	// how to build AppDef with command
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		cmd := wsb.AddCommand(cmdName)
		cmd.SetEngine(appdef.ExtensionEngineKind_WASM)
		cmd.
			SetParam(parName).
			SetResult(resName)
		cmd.SetUnloggedParam(unlName)

		_ = wsb.AddObject(parName)
		_ = wsb.AddObject(unlName)
		_ = wsb.AddObject(resName)

		app = adb.MustBuild()
	}

	// how to enum commands
	{
		cnt := 0
		for c := range appdef.Commands(app.Types) {
			cnt++
			fmt.Println(cnt, c)
		}
		fmt.Println("overall command(s):", cnt)
	}

	// how to inspect builded AppDef with command
	{
		cmd := appdef.Command(app.Type, cmdName)
		fmt.Println(cmd, ":")
		fmt.Println(" - parameter:", cmd.Param())
		fmt.Println(" - unl.param:", cmd.UnloggedParam())
		fmt.Println(" - result   :", cmd.Result())
	}

	// Output:
	// 1 WASM-Command «test.cmd»
	// overall command(s): 1
	// WASM-Command «test.cmd» :
	//  - parameter: Object «test.param»
	//  - unl.param: Object «test.secure»
	//  - result   : Object «test.res»
}
