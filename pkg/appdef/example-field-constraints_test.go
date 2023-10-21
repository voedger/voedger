/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIFieldsBuilder_AddStringField() {

	var app appdef.IAppDef
	docName := appdef.NewQName("test", "doc")

	// how to build doc with string field
	{
		appDef := appdef.New()

		doc := appDef.AddCDoc(docName)
		doc.
			AddStringField("code", true, appdef.MinLen(1), appdef.MaxLen(4), appdef.Pattern(`^\d+$`)).
			SetFieldComment("code", "Code is string containing from one to four digits")

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect string field
	{
		doc := app.CDoc(docName)
		fmt.Printf("doc %q: %v\n", doc.QName(), doc.Kind())
		fmt.Printf("doc field count: %v\n", doc.UserFieldCount())

		f := doc.Field("code")
		fmt.Printf("field %q: kind: %v, required: %v, comment: %s\n", f.Name(), f.DataKind(), f.Required(), f.Comment())

		str := []string{}
		f.Constraints(func(c appdef.IConstraint) {
			str = append(str, fmt.Sprint(c))
		})
		if len(str) > 0 {
			fmt.Printf("  - constraints: [%v]", strings.Join(str, `, `))
		}
	}

	// Output:
	// doc "test.doc": TypeKind_CDoc
	// doc field count: 1
	// field "code": kind: DataKind_string, required: true, comment: Code is string containing from one to four digits
	//   - constraints: [MinLen: 1, MaxLen: 4, Pattern: `^\d+$`]
}

func ExampleIFieldsBuilder_AddBytesField() {

	var app appdef.IAppDef
	docName := appdef.NewQName("test", "doc")

	// how to build doc with bytes field
	{
		appDef := appdef.New()

		doc := appDef.AddCDoc(docName)
		doc.
			AddBytesField("barCode", false, appdef.MaxLen(1024)).
			SetFieldComment("barCode", "Bar code scan data, up to 1 KB")

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect bytes field
	{
		doc := app.CDoc(docName)
		fmt.Printf("doc %q: %v\n", doc.QName(), doc.Kind())
		fmt.Printf("doc field count: %v\n", doc.UserFieldCount())

		f := doc.Field("barCode")
		fmt.Printf("field %q: kind: %v, required: %v, comment: %s\n", f.Name(), f.DataKind(), f.Required(), f.Comment())

		str := []string{}
		f.Constraints(func(c appdef.IConstraint) {
			str = append(str, fmt.Sprint(c))
		})
		if len(str) > 0 {
			fmt.Printf("  - constraints: [%v]", strings.Join(str, `, `))
		}
	}

	// Output:
	// doc "test.doc": TypeKind_CDoc
	// doc field count: 1
	// field "barCode": kind: DataKind_bytes, required: false, comment: Bar code scan data, up to 1 KB
	//   - constraints: [MaxLen: 1024]
}
