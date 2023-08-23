/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIFieldsBuilder_AddStringField() {

	var app appdef.IAppDef
	docName := appdef.NewQName("test", "doc")

	// how to build CDoc with string field
	{
		appDef := appdef.New()

		doc := appDef.AddCDoc(docName)
		doc.
			AddStringField("code", true, appdef.MinLen(1), appdef.MaxLen(4), appdef.Pattern(`^\d+$`)) // from one to four digits

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspected builded CDoc with string field
	{
		doc := app.CDoc(docName)
		fmt.Printf("doc %q: %v\n", doc.QName(), doc.Kind())
		fmt.Printf("doc field count: %v\n", doc.UserFieldCount())

		f := doc.Field("code")
		fmt.Printf("field %q: kind: %v, required: %v\n", f.Name(), f.DataKind(), f.Required())

		if f, ok := f.(appdef.IStringField); ok {
			fmt.Printf("minLen: %d, maxLen: %d, pattern: `%v`", f.Restricts().MinLen(), f.Restricts().MaxLen(), f.Restricts().Pattern())
		}
	}

	// Output:
	// doc "test.doc": DefKind_CDoc
	// doc field count: 1
	// field "code": kind: DataKind_string, required: true
	// minLen: 1, maxLen: 4, pattern: `^\d+$`
}
