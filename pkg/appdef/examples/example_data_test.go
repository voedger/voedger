/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"sort"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
)

func ExampleDataTypes() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	numName := appdef.NewQName("test", "num")
	floatName := appdef.NewQName("test", "float")
	strName := appdef.NewQName("test", "string")
	tokenName := appdef.NewQName("test", "token")
	weekDayName := appdef.NewQName("test", "weekDay")
	jsonName := appdef.NewQName("test", "json")

	// how to build AppDef with data types
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)

		ws.AddData(numName, appdef.DataKind_int64, appdef.NullQName, constraints.MinExcl(0)).SetComment("Natural number")

		_ = ws.AddData(floatName, appdef.DataKind_float64, appdef.NullQName)

		_ = ws.AddData(strName, appdef.DataKind_string, appdef.NullQName, constraints.MinLen(1), constraints.MaxLen(4))

		_ = ws.AddData(tokenName, appdef.DataKind_string, strName, constraints.Pattern("^[a-z]+$"))

		_ = ws.AddData(weekDayName, appdef.DataKind_string, strName, constraints.Enum("Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"))

		ws.AddData(jsonName, appdef.DataKind_string, appdef.NullQName,
			constraints.MaxLen(appdef.MaxFieldLength)).SetComment("JSON string up to 64K")

		app = adb.MustBuild()
	}

	// how to inspect data types in builded AppDef
	{
		cnt := 0
		for d := range appdef.DataTypes(app.Types()) {
			if !d.IsSystem() {
				cnt++
				fmt.Println("-", d, "inherits from", d.Ancestor())
				if d.Comment() != "" {
					fmt.Println(" ", d.Comment())
				}
				str := []string{}
				for _, c := range d.Constraints(false) {
					str = append(str, fmt.Sprint(c))
				}
				if len(str) > 0 {
					sort.Strings(str)
					fmt.Printf("  constraints: (%v)\n", strings.Join(str, `, `))
				}
			}
		}
		fmt.Println("overall user data types: ", cnt)
	}

	// Output:
	// - float64-data «test.float» inherits from float64-data «sys.float64»
	// - string-data «test.json» inherits from string-data «sys.string»
	//   JSON string up to 64K
	//   constraints: (MaxLen: 65535)
	// - int64-data «test.num» inherits from int64-data «sys.int64»
	//   Natural number
	//   constraints: (MinExcl: 0)
	// - string-data «test.string» inherits from string-data «sys.string»
	//   constraints: (MaxLen: 4, MinLen: 1)
	// - string-data «test.token» inherits from string-data «test.string»
	//   constraints: (Pattern: `^[a-z]+$`)
	// - string-data «test.weekDay» inherits from string-data «test.string»
	//   constraints: (Enum: [Fri Mon Sat Sun Thu Tue Wed])
	// overall user data types:  6
}
