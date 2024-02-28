/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIAppDefBuilder_AddPackage() {
	var app appdef.IAppDef

	// how to build AppDef with packages
	{
		appDef := appdef.New()

		appDef.AddPackage("test", "test/path")
		appDef.AddPackage("example", "example/path")

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect builded AppDef with packages
	{
		fmt.Println(app.PackageLocalName("test/path"), app.PackageFullPath("test"))
		fmt.Println(app.PackageLocalName("example/path"), app.PackageFullPath("example"))

		fmt.Println(app.PackageLocalNames())

		app.Packages(func(localName, fullPath string) {
			fmt.Println(localName, fullPath)
		})
	}

	// Output:
	// test test/path
	// example example/path
	// [example test]
	// example example/path
	// test test/path
}
