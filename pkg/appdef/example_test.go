/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func Example() {

	var app appdef.IAppDef
	docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	// how to build AppDef with CDoc
	{
		appDef := appdef.New()

		doc := appDef.AddCDoc(docName)
		doc.SetComment("This is example doc")
		doc.
			AddField("f1", appdef.DataKind_int64, true, "Field may have comments too").
			AddField("f2", appdef.DataKind_string, false)
		rec := appDef.AddCRecord(recName)

		doc.AddContainer("rec", recName, 0, appdef.Occurs_Unbounded)

		rec.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect builded AppDef with CDoc
	{
		// how to find type by name
		t := app.Type(docName)
		fmt.Printf("type %q: %v\n", t.QName(), t.Kind())

		// how to cast type to cdoc
		d, ok := t.(appdef.ICDoc)
		fmt.Printf("%q is CDoc: %v\n", d.QName(), ok && (d.Kind() == appdef.TypeKind_CDoc))

		// how to find CDoc by name
		doc := app.CDoc(docName)
		fmt.Printf("doc %q: %v. %s\n", doc.QName(), doc.Kind(), d.Comment())

		// how to inspect doc fields
		fmt.Printf("doc field count: %v\n", doc.UserFieldCount())

		fmt.Println("founded", doc.Field("f1"))

		fldCnt := 0
		doc.Fields(func(f appdef.IField) {
			fldCnt++
			if f.IsSys() {
				fmt.Print("*")
			} else {
				fmt.Print(" ")
			}
			info := fmt.Sprintf("%d. %v, required: %v", fldCnt, f, f.Required())
			if f.Comment() != "" {
				info += ". " + f.Comment()
			}
			fmt.Println(info)
		})

		// how to inspect doc containers
		fmt.Printf("doc container count: %v\n", doc.ContainerCount())

		fmt.Println("founded", doc.Container("rec"))

		contCnt := 0
		doc.Containers(func(c appdef.IContainer) {
			contCnt++
			fmt.Printf("%d. %v, occurs: %v…%v\n", contCnt, c, c.MinOccurs(), c.MaxOccurs())
		})

		// what if unknown type
		fmt.Println("unknown type:", app.Type(appdef.NewQName("test", "unknown")))
	}

	// Output:
	// type "test.doc": TypeKind_CDoc
	// "test.doc" is CDoc: true
	// doc "test.doc": TypeKind_CDoc. This is example doc
	// doc field count: 2
	// founded int64-field «f1»
	// *1. QName-field «sys.QName», required: true
	// *2. RecordID-field «sys.ID», required: true
	// *3. bool-field «sys.IsActive», required: false
	//  4. int64-field «f1», required: true. Field may have comments too
	//  5. string-field «f2», required: false
	// doc container count: 1
	// founded container «rec: test.rec»
	// 1. container «rec: test.rec», occurs: 0…unbounded
	// unknown type: null type
}
