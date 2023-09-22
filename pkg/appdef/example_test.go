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
		fmt.Printf("%d types\n", app.TypeCount())

		// how to find def by name
		def := app.Type(docName)
		fmt.Printf("type %q: %v\n", def.QName(), def.Kind())

		// how to cast def to cdoc
		d, ok := def.(appdef.ICDoc)
		fmt.Printf("%q is CDoc: %v\n", d.QName(), ok && (d.Kind() == appdef.TypeKind_CDoc))

		// how to find CDoc by name
		doc := app.CDoc(docName)
		fmt.Printf("doc %q: %v. %s\n", doc.QName(), doc.Kind(), d.Comment())

		// how to inspect doc fields
		fmt.Printf("doc field count: %v\n", doc.UserFieldCount())

		f1 := doc.Field("f1")
		fmt.Printf("field %q: kind: %v, required: %v\n", f1.Name(), f1.DataKind(), f1.Required())

		fldCnt := 0
		doc.Fields(func(f appdef.IField) {
			fldCnt++
			if f.IsSys() {
				fmt.Print("*")
			} else {
				fmt.Print(" ")
			}
			info := fmt.Sprintf("%d. Name: %q, kind: %v, required: %v", fldCnt, f.Name(), f.DataKind(), f.Required())
			if f.Comment() != "" {
				info += ". " + f.Comment()
			}
			fmt.Println(info)
		})

		// how to inspect doc containers
		fmt.Printf("doc container count: %v\n", doc.ContainerCount())

		c1 := doc.Container("rec")
		fmt.Printf("container %q: QName: %q, occurs: %v…%v\n", c1.Name(), c1.QName(), c1.MinOccurs(), c1.MaxOccurs())

		contCnt := 0
		doc.Containers(func(c appdef.IContainer) {
			contCnt++
			fmt.Printf("%d. Name: %q, QName: %q, occurs: %v…%v\n", contCnt, c.Name(), c.QName(), c.MinOccurs(), c.MaxOccurs())
		})
	}

	// Output:
	// 2 types
	// type "test.doc": TypeKind_CDoc
	// "test.doc" is CDoc: true
	// doc "test.doc": TypeKind_CDoc. This is example doc
	// doc field count: 2
	// field "f1": kind: DataKind_int64, required: true
	// *1. Name: "sys.QName", kind: DataKind_QName, required: true
	// *2. Name: "sys.ID", kind: DataKind_RecordID, required: true
	// *3. Name: "sys.IsActive", kind: DataKind_bool, required: false
	//  4. Name: "f1", kind: DataKind_int64, required: true. Field may have comments too
	//  5. Name: "f2", kind: DataKind_string, required: false
	// doc container count: 1
	// container "rec": QName: "test.rec", occurs: 0…unbounded
	// 1. Name: "rec", QName: "test.rec", occurs: 0…unbounded
}
