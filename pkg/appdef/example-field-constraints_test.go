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

		doc := appDef.AddCDoc(docName)
		doc.
			AddField("code", appdef.DataKind_string, true, appdef.MinLen(1), appdef.MaxLen(4), appdef.Pattern(`^\d+$`)).
			SetFieldComment("code", "Code is string containing from one to four digits").
			AddField("barCode", appdef.DataKind_bytes, false, appdef.MaxLen(1024)).
			SetFieldComment("barCode", "Bar code scan data, up to 1 KB")

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect fields
	{
		doc := app.CDoc(docName)
		fmt.Printf("doc %q: %v\n", doc.QName(), doc.Kind())
		fmt.Printf("doc field count: %v\n", doc.UserFieldCount())

		cnt := 0
		doc.UserFields(func(f appdef.IField) {
			cnt++
			fmt.Printf("%d. %v, required: %v, comment: %s\n", cnt, f, f.Required(), f.Comment())
			str := []string{}
			f.Constraints(func(c appdef.IConstraint) {
				str = append(str, fmt.Sprint(c))
			})
			if len(str) > 0 {
				fmt.Printf("  - constraints: [%v]\n", strings.Join(str, `, `))
			}
		})

	}

	// Output:
	// doc "test.doc": TypeKind_CDoc
	// doc field count: 2
	// 1. string-field «code», required: true, comment: Code is string containing from one to four digits
	//   - constraints: [MinLen: 1, MaxLen: 4, Pattern: `^\d+$`]
	// 2. bytes-field «barCode», required: false, comment: Bar code scan data, up to 1 KB
	//   - constraints: [MaxLen: 1024]
}
