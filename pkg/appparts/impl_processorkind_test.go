/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/sys"
)

func TestProcessorKind_compatibleWithExtension(t *testing.T) {
	cmd := appdef.NewQName("test", "cmd")
	query := appdef.NewQName("test", "query")
	syncPrj := appdef.NewQName("test", "syncPrj")
	asyncPrj := appdef.NewQName("test", "asyncPrj")
	job := appdef.NewQName("test", "job")

	names := appdef.QNamesFrom(cmd, query, syncPrj, asyncPrj, job)

	appDef := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.test/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		wsb.AddCommand(cmd).SetParam(appdef.QNameAnyObject)

		wsb.AddQuery(query).SetResult(appdef.QNameAnyView)

		p1 := wsb.AddProjector(syncPrj)
		p1.SetSync(true)
		p1.States().Add(sys.Storage_Record, appdef.QNameAnyRecord)
		p1.Intents().Add(sys.Storage_View, appdef.QNameAnyView)
		p1.Events().Add(cmd)

		p2 := wsb.AddProjector(asyncPrj)
		p2.SetSync(false)
		p2.States().Add(sys.Storage_Record, appdef.QNameAnyRecord)
		p2.Intents().Add(sys.Storage_View, appdef.QNameAnyView)
		p2.Events().Add(cmd)

		wsb.AddJob(job).SetCronSchedule("@every 1m")

		return adb.MustBuild()
	}()

	tests := []struct {
		proc        ProcessorKind
		compatibles appdef.QNames
	}{
		{ProcessorKind_Command, appdef.QNamesFrom(cmd, syncPrj)},
		{ProcessorKind_Query, appdef.QNamesFrom(query)},
		{ProcessorKind_Actualizer, appdef.QNamesFrom(asyncPrj)},
		{ProcessorKind_Scheduler, appdef.QNamesFrom(job)},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("compatibles for %v", tt.proc), func(t *testing.T) {
			for _, n := range names {
				ext := appdef.Extension(appDef.Type, n)
				got, _ := tt.proc.compatibleWithExtension(ext)
				if want := tt.compatibles.Contains(n); got != want {
					t.Errorf("%v.compatibleWithExtension(%v) = %v, want %v", tt.proc, ext, got, want)
				}
			}
		})
	}
}
