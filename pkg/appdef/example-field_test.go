/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIFieldsBuilder_AddField() {

	var app appdef.IAppDef
	docName := appdef.NewQName("test", "doc")

	// how to build doc with string field
	{
		appDef := appdef.New()

		doc := appDef.AddODoc(docName)
		doc.
			AddField("code", appdef.DataKind_string, true, appdef.MinLen(1), appdef.MaxLen(4), appdef.Pattern(`^\d+$`)).
			SetFieldComment("code", "Code is string containing from one to four digits").
			AddField("barCode", appdef.DataKind_bytes, false, appdef.MaxLen(4096)).
			SetFieldComment("barCode", "Bar code scan data, up to 4 KB")

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect fields
	{
		doc := app.ODoc(docName)
		fmt.Printf("%v, user field count: %v\n", doc, doc.UserFieldCount())

		cnt := 0
		doc.UserFields(func(f appdef.IField) {
			cnt++
			fmt.Printf("%d. %v", cnt, f)
			if f.Required() {
				fmt.Print(", required")
			}
			if c := f.Comment(); c != "" {
				fmt.Print(". ", c)
			}
			str := []string{}
			f.Constraints(func(c appdef.IConstraint) {
				str = append(str, fmt.Sprint(c))
			})
			if len(str) > 0 {
				fmt.Println()
				fmt.Printf("  - constraints: [%v]", strings.Join(str, `, `))
			}
			fmt.Println()
		})
	}

	// Output:
	// ODoc «test.doc», user field count: 2
	// 1. string-field «code», required. Code is string containing from one to four digits
	//   - constraints: [MinLen: 1, MaxLen: 4, Pattern: `^\d+$`]
	// 2. bytes-field «barCode». Bar code scan data, up to 4 KB
	//   - constraints: [MaxLen: 4096]
}

func ExampleIFieldsBuilder_AddDataField() {

	var app appdef.IAppDef
	docName := appdef.NewQName("test", "doc")

	// how to build doc with string field
	{
		appDef := appdef.New()

		str10 := appDef.AddData(appdef.NewQName("test", "str10"), appdef.DataKind_string, appdef.NullQName, appdef.MinLen(10), appdef.MaxLen(10))
		str10.SetComment("String with 10 characters exact")
		dig10 := appDef.AddData(appdef.NewQName("test", "dig10"), appdef.DataKind_string, str10.QName(), appdef.Pattern(`^\d+$`, "only digits"))

		month := appDef.AddData(appdef.NewQName("test", "month"), appdef.DataKind_int32, appdef.NullQName, appdef.MinExcl(0), appdef.MaxIncl(12))
		month.SetComment("Month number, left-open range (0-12]")

		doc := appDef.AddCDoc(docName)
		doc.
			AddDataField("code", dig10.QName(), true).
			SetFieldComment("code", "Code is string containing 10 digits").
			AddDataField("month", month.QName(), true).
			SetFieldComment("month", "Month number natural up to 12")

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect fields
	{
		doc := app.CDoc(docName)
		fmt.Printf("%v, user field count: %v\n", doc, doc.UserFieldCount())

		cnt := 0
		doc.UserFields(func(f appdef.IField) {
			cnt++
			fmt.Printf("%d. %v", cnt, f)
			if f.Required() {
				fmt.Print(", required")
			}
			if c := f.Comment(); c != "" {
				fmt.Print(". ", c)
			}
			str := []string{}
			f.Constraints(func(c appdef.IConstraint) {
				str = append(str, fmt.Sprint(c))
			})
			if len(str) > 0 {
				fmt.Println()
				fmt.Printf("  - constraints: [%v]", strings.Join(str, `, `))
			}
			fmt.Println()
		})

	}

	// Output:
	// CDoc «test.doc», user field count: 2
	// 1. string-field «code», required. Code is string containing 10 digits
	//   - constraints: [MinLen: 10, MaxLen: 10, Pattern: `^\d+$`]
	// 2. int32-field «month», required. Month number natural up to 12
	//   - constraints: [MinExcl: 0, MaxIncl: 12]
}

func ExampleIFieldsBuilder_SetFieldVerify() {

	var app appdef.IAppDef
	docName := appdef.NewQName("test", "doc")

	// how to build doc with verified field
	{
		appDef := appdef.New()

		doc := appDef.AddCDoc(docName)
		doc.
			AddField("pin", appdef.DataKind_string, true, appdef.MinLen(4), appdef.MaxLen(4), appdef.Pattern(`^\d+$`)).
			SetFieldComment("pin", "Secret four digits pin code").
			SetFieldVerify("pin", appdef.VerificationKind_EMail, appdef.VerificationKind_Phone)

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect verified field
	{
		doc := app.CDoc(docName)
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
