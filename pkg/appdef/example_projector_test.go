/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIAppDefBuilder_AddProjector() {

	var app appdef.IAppDef

	sysRecords, sysViews := appdef.NewQName(appdef.SysPackage, "records"), appdef.NewQName(appdef.SysPackage, "views")

	prjName := appdef.NewQName("test", "projector")
	recName := appdef.NewQName("test", "record")
	docName := appdef.NewQName("test", "doc")
	viewName := appdef.NewQName("test", "view")

	// how to build AppDef with projectors
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		adb.AddCRecord(recName).SetComment("record is trigger for projector")
		adb.AddCDoc(docName).SetComment("doc is state for projector")

		v := adb.AddView(viewName)
		v.Key().PartKey().AddDataField("id", appdef.SysData_RecordID)
		v.Key().ClustCols().AddDataField("name", appdef.SysData_String)
		v.Value().AddDataField("data", appdef.SysData_bytes, false, appdef.MaxLen(1024))
		v.SetComment("view is intent for projector")

		prj := adb.AddProjector(prjName)
		prj.SetWantErrors()
		prj.Events().
			Add(recName, appdef.ProjectorEventKind_AnyChanges...).
			SetComment(recName, fmt.Sprintf("run projector every time when %v is changed", recName))
		prj.States().
			Add(sysRecords, docName).
			SetComment(sysRecords, "projector needs to read «test.doc» from «sys.records» storage")
		prj.Intents().
			Add(sysViews, viewName).
			SetComment(sysViews, "projector needs to update «test.view» from «sys.views» storage")

		app = adb.MustBuild()
	}

	// how to inspect builded AppDef with projector
	{
		prj := app.Projector(prjName)
		fmt.Println(prj, ":")
		prj.Events().Enum(func(e appdef.IProjectorEvent) {
			fmt.Println(" - event:", e, e.Comment())
		})
		if prj.WantErrors() {
			fmt.Println(" - want sys.error events")
		}
		for s := range prj.States().Enum {
			fmt.Println(" - state:", s, s.Comment())
		}
		for i := range prj.Intents().Enum {
			fmt.Println(" - intent:", i, i.Comment())
		}

		fmt.Println(app.Projector(appdef.NewQName("test", "unknown")))
	}

	// How to enum all projectors in AppDef
	{
		cnt := 0
		for prj := range app.Projectors {
			cnt++
			fmt.Println(cnt, prj)
		}
	}

	// Output:
	// BuiltIn-Projector «test.projector» :
	//  - event: CRecord «test.record» [Insert Update Activate Deactivate] run projector every time when test.record is changed
	//  - want sys.error events
	//  - state: Storage «sys.records» [test.doc] projector needs to read «test.doc» from «sys.records» storage
	//  - intent: Storage «sys.views» [test.view] projector needs to update «test.view» from «sys.views» storage
	// <nil>
	// 1 BuiltIn-Projector «test.projector»
}
