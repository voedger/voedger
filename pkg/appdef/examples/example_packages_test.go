/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
)

func ExampleIAppDefBuilder_AddPackage() {
	var app appdef.IAppDef

	// how to build AppDef with packages
	{
		adb := builder.New()

		adb.AddPackage("test", "test.com/test")
		adb.AddPackage("example", "example.com/example")

		app = adb.MustBuild()
	}

	// how to inspect builded AppDef with packages
	{
		fmt.Println(app.PackageLocalName("test.com/test"), app.PackageFullPath("test"))
		fmt.Println(app.PackageLocalName("example.com/example"), app.PackageFullPath("example"))

		fmt.Println(app.PackageLocalNames())

		fmt.Println(app.FullQName(appdef.NewQName("test", "name")))
		fmt.Println(app.LocalQName(appdef.NewFullQName("example.com/example", "name")))
	}

	// Output:
	// test test.com/test
	// example example.com/example
	// [example sys test]
	// test.com/test.name
	// example.name
}
