/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/sys"
)

func TestProcessorKind_CompatibleWithExtension(t *testing.T) {
	cmd := appdef.NewQName("test", "cmd")
	query := appdef.NewQName("test", "query")
	syncPrj := appdef.NewQName("test", "syncPrj")
	asyncPrj := appdef.NewQName("test", "asyncPrj")
	job := appdef.NewQName("test", "job")

	names := appdef.QNamesFrom(cmd, query, syncPrj, asyncPrj, job)

	appDef := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test.test/test")

		wsName := appdef.NewQName("test", "workspace")
		wsb := adb.AddWorkspace(wsName)

		wsb.AddCommand(cmd).SetParam(appdef.QNameAnyObject)

		wsb.AddQuery(query).SetResult(appdef.QNameAnyView)

		p1 := wsb.AddProjector(syncPrj)
		p1.Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.Types(wsName, appdef.TypeKind_Command))
		p1.SetSync(true)
		p1.States().Add(sys.Storage_Record, appdef.QNameAnyRecord)
		p1.Intents().Add(sys.Storage_View, appdef.QNameAnyView)

		p2 := wsb.AddProjector(asyncPrj)
		p2.Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.Types(wsName, appdef.TypeKind_Command))
		p2.SetSync(false)
		p2.States().Add(sys.Storage_Record, appdef.QNameAnyRecord)
		p2.Intents().Add(sys.Storage_View, appdef.QNameAnyView)

		wsb.AddJob(job).SetCronSchedule("@every 1m")

		return adb.MustBuild()
	}()

	tests := []struct {
		proc        appparts.ProcessorKind
		compatibles appdef.QNames
	}{
		{appparts.ProcessorKind_Command, appdef.QNamesFrom(cmd, syncPrj)},
		{appparts.ProcessorKind_Query, appdef.QNamesFrom(query)},
		{appparts.ProcessorKind_Actualizer, appdef.QNamesFrom(asyncPrj)},
		{appparts.ProcessorKind_Scheduler, appdef.QNamesFrom(job)},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("compatibles for %v", tt.proc), func(t *testing.T) {
			for _, n := range names {
				ext := appdef.Extension(appDef.Type, n)
				got, _ := tt.proc.CompatibleWithExtension(ext)
				if want := tt.compatibles.Contains(n); got != want {
					t.Errorf("%v.compatibleWithExtension(%v) = %v, want %v", tt.proc, ext, got, want)
				}
			}
		})
	}
}
