/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
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
		appDef := appdef.New()

		appDef.AddCRecord(recName).SetComment("record is trigger for projector")
		appDef.AddCDoc(docName).SetComment("doc is state for projector")

		v := appDef.AddView(viewName)
		v.KeyBuilder().PartKeyBuilder().AddDataField("id", appdef.SysData_RecordID)
		v.KeyBuilder().ClustColsBuilder().AddDataField("name", appdef.SysData_String)
		v.ValueBuilder().AddDataField("data", appdef.SysData_bytes, false, appdef.MaxLen(1024))
		v.SetComment("view is intent for projector")

		prj := appDef.AddProjector(prjName)
		prj.
			AddEvent(recName, appdef.ProjectorEventKind_AnyChanges...).
			SetEventComment(recName, fmt.Sprintf("run projector every time when %v is changed", recName)).
			AddState(sysRecords, docName).
			AddIntent(sysViews, viewName)

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspect builded AppDef with projector
	{
		prj := app.Projector(prjName)
		fmt.Println(prj, ":")
		prj.Events(func(e appdef.IProjectorEvent) {
			fmt.Println(" - event:", e, e.Comment())
		})
		prj.States(func(s appdef.QName, names appdef.QNames) {
			fmt.Println(" - state:", s, names)
		})
		prj.Intents(func(s appdef.QName, names appdef.QNames) {
			fmt.Println(" - intent:", s, names)
		})

		fmt.Println(app.Projector(appdef.NewQName("test", "unknown")))
	}

	// How to enum all projectors in AppDef
	{
		cnt := 0
		app.Projectors(func(prj appdef.IProjector) {
			cnt++
			fmt.Println(cnt, prj)
		})
	}

	// Output:
	// BuiltIn-Projector «test.projector» :
	//  - event: CRecord «test.record» [Insert Update Activate Deactivate] run projector every time when test.record is changed
	//  - state: sys.records [test.doc]
	//  - intent: sys.views [test.view]
	// <nil>
	// 1 BuiltIn-Projector «test.projector»
}
