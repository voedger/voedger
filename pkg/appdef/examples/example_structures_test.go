/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
)

func ExampleStructures() {

	var app appdef.IAppDef
	docName, recName := appdef.NewQName("test", "document"), appdef.NewQName("test", "record")
	objName, childName := appdef.NewQName("test", "object"), appdef.NewQName("test", "child")

	// how to build AppDef with structures
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		doc := wsb.AddCDoc(docName)
		doc.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)
		doc.AddContainer("rec", recName, 0, appdef.Occurs_Unbounded)

		rec := wsb.AddCRecord(recName)
		rec.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)

		obj := wsb.AddObject(objName)
		obj.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)
		obj.AddContainer("child", childName, 0, appdef.Occurs_Unbounded)

		child := wsb.AddObject(childName)
		child.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)

		app = adb.MustBuild()
	}

	// how to inspect builded AppDef with structures
	{
		// how to enum structures
		cnt := 0
		for s := range appdef.Structures(app.Types()) {
			if s.IsSystem() {
				continue // skip system structures
			}
			cnt++
			fmt.Printf("%d. %v\n", cnt, s)
			fmt.Printf("- user/overall field count: %d/%d\n", s.UserFieldCount(), s.FieldCount())
			fmt.Printf("- container count: %d\n", s.ContainerCount())
		}
		fmt.Printf("Overall %d structures\n", cnt)

		// how to find structure by name
		fmt.Println(appdef.Structure(app.Type, docName))
		fmt.Println(appdef.Structure(app.Type, appdef.NewQName("test", "unknown")))
	}

	// how to inspect builded AppDef with records
	{
		cnt := 0
		for r := range appdef.Records(app.Types()) {
			if r.IsSystem() {
				continue // skip system structures
			}
			cnt++
			fmt.Printf("%d. %v\n", cnt, r)
			fmt.Printf("- user/overall field count: %d/%d\n", r.UserFieldCount(), r.FieldCount())
			fmt.Printf("- container count: %d\n", r.ContainerCount())
		}

		fmt.Printf("Overall %d records\n", cnt)

		fmt.Println(appdef.Record(app.Type, recName), appdef.Record(app.Type, appdef.NewQName("test", "unknown")))
	}

	// Output:
	// 1. Object «test.child»
	// - user/overall field count: 2/4
	// - container count: 0
	// 2. CDoc «test.document»
	// - user/overall field count: 2/5
	// - container count: 1
	// 3. Object «test.object»
	// - user/overall field count: 2/4
	// - container count: 1
	// 4. CRecord «test.record»
	// - user/overall field count: 2/7
	// - container count: 0
	// Overall 4 structures
	// CDoc «test.document»
	// <nil>
	// 1. CDoc «test.document»
	// - user/overall field count: 2/5
	// - container count: 1
	// 2. CRecord «test.record»
	// - user/overall field count: 2/7
	// - container count: 0
	// Overall 2 records
	// CRecord «test.record» <nil>
}
