/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package constraints_test

import (
	"fmt"
	"sort"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
)

func ExampleMinLen() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	strName := appdef.NewQName("test", "string")

	// how to build AppDef with data types constrainted with MinLen
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)

		_ = ws.AddData(strName, appdef.DataKind_string, appdef.NullQName, constraints.MinLen(1))

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
	// - string-data «test.string» inherits from string-data «sys.string»
	//   constraints: (MinLen: 1)
	// overall user data types:  1
}

func ExampleMaxLen() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	strName := appdef.NewQName("test", "string")

	// how to build AppDef with data types constrainted with MaxLen
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)

		_ = ws.AddData(strName, appdef.DataKind_string, appdef.NullQName, constraints.MaxLen(4))

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
	// - string-data «test.string» inherits from string-data «sys.string»
	//   constraints: (MaxLen: 4)
	// overall user data types:  1
}

func ExamplePattern() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	strName := appdef.NewQName("test", "string")

	// how to build AppDef with data types constrainted with Pattern
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)

		_ = ws.AddData(strName, appdef.DataKind_string, appdef.NullQName, constraints.Pattern("^[a-z]+$"))

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
	// - string-data «test.string» inherits from string-data «sys.string»
	//   constraints: (Pattern: `^[a-z]+$`)
	// overall user data types:  1
}

func ExampleMinIncl() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	floatName := appdef.NewQName("test", "float")

	// how to build AppDef with data types constrainted with MinIncl
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)

		ws.AddData(floatName, appdef.DataKind_float64, appdef.NullQName, constraints.MinIncl(0)).SetComment("Nonnegative float")

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
	//   Nonnegative float
	//   constraints: (MinIncl: 0)
	// overall user data types:  1
}

func ExampleMinExcl() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	naturalName := appdef.NewQName("test", "natural")

	// how to build AppDef with data types constrainted with MinExcl
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)

		ws.AddData(naturalName, appdef.DataKind_int64, appdef.NullQName, constraints.MinExcl(0)).SetComment("Natural number")

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
	// - int64-data «test.natural» inherits from int64-data «sys.int64»
	//   Natural number
	//   constraints: (MinExcl: 0)
	// overall user data types:  1
}

func ExampleMaxIncl() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	floatName := appdef.NewQName("test", "float")

	// how to build AppDef with data types constrainted with MaxIncl
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)

		_ = ws.AddData(floatName, appdef.DataKind_float64, appdef.NullQName, constraints.MaxIncl(100))

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
	//   constraints: (MaxIncl: 100)
	// overall user data types:  1
}

func ExampleMaxExcl() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	intName := appdef.NewQName("test", "int")

	// how to build AppDef with data types constrainted with MaxExcl
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)

		_ = ws.AddData(intName, appdef.DataKind_int64, appdef.NullQName, constraints.MaxExcl(100))

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
	// - int64-data «test.int» inherits from int64-data «sys.int64»
	//   constraints: (MaxExcl: 100)
	// overall user data types:  1
}

func ExampleEnum() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	weekdayName := appdef.NewQName("test", "weekday")

	// how to build AppDef with data types constrainted with MaxExcl
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		ws := adb.AddWorkspace(wsName)

		ws.AddData(weekdayName, appdef.DataKind_string, appdef.NullQName, constraints.Enum("Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun")).SetComment("Week day")

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
	// - string-data «test.weekday» inherits from string-data «sys.string»
	//   Week day
	//   constraints: (Enum: [Fri Mon Sat Sun Thu Tue Wed])
	// overall user data types:  1
}
