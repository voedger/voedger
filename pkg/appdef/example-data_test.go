/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIAppDefBuilder_AddData() {

	var app appdef.IAppDef

	intName := appdef.NewQName("test", "int")
	floatName := appdef.NewQName("test", "float")
	strName := appdef.NewQName("test", "string")
	tokenName := appdef.NewQName("test", "token")

	// how to build AppDef with data types
	{
		appDef := appdef.New()

		_ = appDef.AddData(intName, appdef.DataKind_int64, appdef.NullQName)
		_ = appDef.AddData(floatName, appdef.DataKind_float64, appdef.NullQName)

		s := appDef.AddData(strName, appdef.DataKind_string, appdef.NullQName)
		s.AddConstraints(appdef.MinLen(1), appdef.MaxLen(4))

		t := appDef.AddData(tokenName, appdef.DataKind_string, strName)
		t.AddConstraints(appdef.Pattern("^[a-z]+$"))

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect data types in builded AppDef
	{
		cnt := 0
		app.DataTypes(false, func(d appdef.IData) {
			cnt++
			fmt.Println("-", d, "inherits from", d.Ancestor())
			if d.Constraints().Count() > 0 {
				fmt.Println("  constraints:", d.Constraints())
			}
		})
		fmt.Println("overall data types: ", cnt)
	}

	// Output:
	// - float64-data «test.float» inherits from float64-data «sys.float64»
	// - int64-data «test.int» inherits from int64-data «sys.int64»
	// - string-data «test.string» inherits from string-data «sys.string»
	//   constraints: MinLen: 1, MaxLen: 4
	// - string-data «test.token» inherits from string-data «test.string»
	//   constraints: Pattern: `^[a-z]+$`
	// overall data types:  4
}
