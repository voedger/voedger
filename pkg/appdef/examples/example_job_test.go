/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
)

func ExampleJobs() {

	var app appdef.IAppDef

	sysViews := appdef.NewQName(appdef.SysPackage, "views")
	viewName := appdef.NewQName("test", "view")
	jobName := appdef.NewQName("test", "job")

	// how to build AppDef with jobs
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		v := wsb.AddView(viewName)
		v.Key().PartKey().AddDataField("id", appdef.SysData_RecordID)
		v.Key().ClustCols().AddDataField("name", appdef.SysData_String)
		v.Value().AddDataField("data", appdef.SysData_bytes, false, constraints.MaxLen(1024))
		v.SetComment("view is state for job")

		job := wsb.AddJob(jobName)
		job.SetCronSchedule(`@every 2m30s`)
		job.States().
			Add(sysViews, viewName).
			SetComment(sysViews, "job needs to read «test.view from «sys.views» storage")

		app = adb.MustBuild()
	}

	// how to find job in builded AppDef
	{
		job := appdef.Job(app.Type, jobName)
		fmt.Println(job, ":")
		fmt.Println(" - crone:", job.CronSchedule())
		for _, n := range job.States().Names() {
			s := job.States().Storage(n)
			fmt.Println(" - state:", s, s.Comment())
		}

		fmt.Println(appdef.Job(app.Type, appdef.NewQName("test", "unknown")))
	}

	// How to enum all jobs in AppDef
	{
		cnt := 0
		for j := range appdef.Jobs(app.Types()) {
			cnt++
			fmt.Println(cnt, j)
		}
	}

	// Output:
	// BuiltIn-Job «test.job» :
	//  - crone: @every 2m30s
	//  - state: Storage «sys.views» [test.view] job needs to read «test.view from «sys.views» storage
	// <nil>
	// 1 BuiltIn-Job «test.job»
}
