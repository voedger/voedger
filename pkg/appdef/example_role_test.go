/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIAppDefBuilder_AddRole() {

	var app appdef.IAppDef

	admRoleName := appdef.NewQName("test", "admRole")
	waiterRoleName := appdef.NewQName("test", "waiterRole")

	// how to build AppDef with roles
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		_ = adb.AddRole(admRoleName)
		_ = adb.AddRole(waiterRoleName)

		app = adb.MustBuild()
	}

	// how to enum roles
	{
		cnt := 0
		app.Roles(func(r appdef.IRole) {
			cnt++
			fmt.Println(cnt, r)
		})
		fmt.Println("overall:", cnt)
	}

	// how to inspect builded AppDef with roles
	{
		r := app.Role(admRoleName)
		fmt.Println(r, ":")

		fmt.Println("Unknown role :", app.Role(appdef.NewQName("test", "unknownRole")))
	}

	// Output:
	// 1 Role «test.admRole»
	// 2 Role «test.waiterRole»
	// overall: 2
	// Role «test.admRole» :
	// Unknown role : <nil>
}
