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

func ExampleIWithFields() {

	var app appdef.IAppDef
	docName := appdef.NewQName("test", "doc")

	// how to build doc with string field
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		doc := wsb.AddODoc(docName)
		doc.
			AddField("code", appdef.DataKind_string, true, constraints.MinLen(1), constraints.MaxLen(4), constraints.Pattern(`^\d+$`)).
			SetFieldComment("code", "Code is string containing from one to four digits").
			AddField("barCode", appdef.DataKind_bytes, false, constraints.MaxLen(4096)).
			SetFieldComment("barCode", "Bar code scan data, up to 4 KB")

		app = adb.MustBuild()
	}

	// how to inspect fields
	{
		doc := appdef.ODoc(app.Type, docName)
		fmt.Printf("%v, user field count: %v\n", doc, doc.UserFieldCount())

		cnt := 0
		for _, f := range doc.Fields() {
			if f.IsSys() {
				continue
			}
			cnt++
			fmt.Printf("%d. %v", cnt, f)
			if f.Required() {
				fmt.Print(", required")
			}
			if c := f.Comment(); c != "" {
				fmt.Print(". ", c)
			}
			str := []string{}
			for _, c := range f.Constraints() {
				str = append(str, fmt.Sprint(c))
			}
			if len(str) > 0 {
				sort.Strings(str)
				fmt.Println()
				fmt.Printf("  - constraints: [%v]", strings.Join(str, `, `))
			}
			fmt.Println()
		}
	}

	// Output:
	// ODoc «test.doc», user field count: 2
	// 1. string-field «code», required. Code is string containing from one to four digits
	//   - constraints: [MaxLen: 4, MinLen: 1, Pattern: `^\d+$`]
	// 2. bytes-field «barCode». Bar code scan data, up to 4 KB
	//   - constraints: [MaxLen: 4096]
}

func ExampleIFieldsBuilder_AddDataField() {

	var app appdef.IAppDef
	docName := appdef.NewQName("test", "doc")

	// how to build doc with string field
	{

		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		str10name := appdef.NewQName("test", "str10")
		str10 := wsb.AddData(str10name, appdef.DataKind_string, appdef.NullQName, constraints.MinLen(10), constraints.MaxLen(10))
		str10.SetComment("String with 10 characters exact")

		dig10name := appdef.NewQName("test", "dig10")
		_ = wsb.AddData(dig10name, appdef.DataKind_string, str10name, constraints.Pattern(`^\d+$`, "only digits"))

		monthName := appdef.NewQName("test", "month")
		month := wsb.AddData(monthName, appdef.DataKind_int32, appdef.NullQName, constraints.MinExcl(0), constraints.MaxIncl(12))
		month.SetComment("Month number, left-open range (0-12]")

		doc := wsb.AddCDoc(docName)
		doc.
			AddDataField("code", dig10name, true).
			SetFieldComment("code", "Code is string containing 10 digits").
			AddDataField("month", monthName, true).
			SetFieldComment("month", "Month number natural up to 12")

		app = adb.MustBuild()
	}

	// how to inspect fields
	{
		doc := appdef.CDoc(app.Type, docName)
		fmt.Printf("%v, user field count: %v\n", doc, doc.UserFieldCount())

		cnt := 0
		for _, f := range doc.Fields() {
			if f.IsSys() {
				continue
			}
			cnt++
			fmt.Printf("%d. %v", cnt, f)
			if f.Required() {
				fmt.Print(", required")
			}
			if c := f.Comment(); c != "" {
				fmt.Print(". ", c)
			}
			str := []string{}
			for _, c := range f.Constraints() {
				str = append(str, fmt.Sprint(c))
			}
			if len(str) > 0 {
				fmt.Println()
				sort.Strings(str)
				fmt.Printf("  - constraints: [%v]", strings.Join(str, `, `))
			}
			fmt.Println()
		}
	}

	// Output:
	// CDoc «test.doc», user field count: 2
	// 1. string-field «code», required. Code is string containing 10 digits
	//   - constraints: [MaxLen: 10, MinLen: 10, Pattern: `^\d+$`]
	// 2. int32-field «month», required. Month number natural up to 12
	//   - constraints: [MaxIncl: 12, MinExcl: 0]
}

func ExampleIFieldsBuilder_SetFieldVerify() {

	var app appdef.IAppDef
	docName := appdef.NewQName("test", "doc")

	// how to build doc with verified field
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		doc := ws.AddCDoc(docName)
		doc.
			AddField("pin", appdef.DataKind_string, true, constraints.MinLen(4), constraints.MaxLen(4), constraints.Pattern(`^\d+$`)).
			SetFieldComment("pin", "Secret four digits pin code").
			SetFieldVerify("pin", appdef.VerificationKind_EMail, appdef.VerificationKind_Phone)

		app = adb.MustBuild()
	}

	// how to inspect verified field
	{
		doc := appdef.CDoc(app.Type, docName)
		fmt.Printf("doc %q: %v\n", doc.QName(), doc.Kind())
		fmt.Printf("doc field count: %v\n", doc.UserFieldCount())

		f := doc.Field("pin")
		fmt.Printf("field %q: kind: %v, required: %v, comment: %s\n", f.Name(), f.DataKind(), f.Required(), f.Comment())
		for v := appdef.VerificationKind_EMail; v < appdef.VerificationKind_FakeLast; v++ {
			fmt.Println(v, f.VerificationKind(v))
		}
	}

	// Output:
	// doc "test.doc": TypeKind_CDoc
	// doc field count: 1
	// field "pin": kind: DataKind_string, required: true, comment: Secret four digits pin code
	// VerificationKind_EMail true
	// VerificationKind_Phone true
}
