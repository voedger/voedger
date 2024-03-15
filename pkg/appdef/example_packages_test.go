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

		appDef.AddPackage("test", "test.com/test")
		appDef.AddPackage("example", "example.com/example")

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect builded AppDef with packages
	{
		fmt.Println(app.PackageLocalName("test.com/test"), app.PackageFullPath("test"))
		fmt.Println(app.PackageLocalName("example.com/example"), app.PackageFullPath("example"))

		fmt.Println(app.PackageLocalNames())

		app.Packages(func(localName, fullPath string) {
			fmt.Println(localName, fullPath)
		})

		fmt.Println(app.FullQName(appdef.NewQName("test", "name")))
		fmt.Println(app.LocalQName(appdef.NewFullQName("example.com/example", "name")))
	}

	// Output:
	// test test.com/test
	// example example.com/example
	// [example test]
	// example example.com/example
	// test test.com/test
	// test.com/test.name
	// example.name
}
