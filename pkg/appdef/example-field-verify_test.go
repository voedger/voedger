/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIFieldsBuilder_SetFieldVerify() {

	var app appdef.IAppDef
	docName := appdef.NewQName("test", "doc")

	// how to build doc with verified field
	{
		appDef := appdef.New()

		doc := appDef.AddCDoc(docName)
		doc.
			AddStringField("pin", true, appdef.MinLen(4), appdef.MaxLen(4), appdef.Pattern(`^\d+$`)).
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
	// doc "test.doc": DefKind_CDoc
	// doc field count: 1
	// field "pin": kind: DataKind_string, required: true, comment: Secret four digits pin code
	// VerificationKind_EMail true
	// VerificationKind_Phone true
}
