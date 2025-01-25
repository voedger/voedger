/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Jobs(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")

	sysViews := appdef.NewQName(appdef.SysPackage, "views")
	resultName := appdef.NewQName("test", "result")
	cronSchedule := `@every 2m30s`
	jobName := appdef.NewQName("test", "job")

	t.Run("should be ok to add job", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		result := wsb.AddView(resultName)
		result.Key().PartKey().AddDataField("id", appdef.SysData_RecordID)
		result.Key().ClustCols().AddDataField("name", appdef.SysData_String)
		result.Value().AddDataField("data", appdef.SysData_bytes, false, constraints.MaxLen(1024))
		result.SetComment("result is intent for job")

		job := wsb.AddJob(jobName)

		job.SetCronSchedule(cronSchedule)
		// #2810 Job may have intents
		job.Intents().Add(sysViews, resultName)

		t.Run("should be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	t.Run("should be ok to find builded job", func(t *testing.T) {
		typ := app.Type(jobName)
		require.Equal(appdef.TypeKind_Job, typ.Kind())

		j, ok := typ.(appdef.IJob)
		require.True(ok)
		require.Equal(appdef.TypeKind_Job, j.Kind())

		job := appdef.Job(app.Type, jobName)
		require.Equal(appdef.TypeKind_Job, job.Kind())
		require.Equal(wsName, job.Workspace().QName())
		require.Equal(j, job)

		require.Equal(jobName.Entity(), job.Name())
		require.Equal(appdef.ExtensionEngineKind_BuiltIn, job.Engine())

		require.Equal(cronSchedule, job.CronSchedule())

		// #2810 Job may have intents
		t.Run("should be ok enum intents", func(t *testing.T) {
			cnt := 0
			for _, n := range job.Intents().Names() {
				cnt++
				i := job.Intents().Storage(n)
				require.Equal(n, i.Name())
				switch cnt {
				case 1:
					require.Equal(sysViews, n)
					require.EqualValues([]appdef.QName{resultName}, i.Names())
				default:
					require.Failf("unexpected intent", "intent: %v", i)
				}
			}
			require.Equal(1, cnt)

			t.Run("should be ok to get intent by name", func(t *testing.T) {
				intent := job.Intents().Storage(sysViews)
				require.NotNil(intent)
				require.Equal(sysViews, intent.Name())
				require.EqualValues([]appdef.QName{resultName}, intent.Names())

				require.Nil(job.Intents().Storage(appdef.NewQName("test", "unknown")), "should be nil for unknown intent")
			})
		})
	})

	t.Run("should be ok to enum jobs", func(t *testing.T) {
		cnt := 0
		for j := range appdef.Jobs(app.Types()) {
			if j.IsSystem() {
				continue
			}
			cnt++
			switch cnt {
			case 1:
				require.Equal(appdef.TypeKind_Job, j.Kind())
				require.Equal(jobName, j.QName())
			default:
				require.Failf("unexpected job", "job: %v", j)
			}
		}
		require.Equal(1, cnt)
	})

	require.Nil(appdef.Job(app.Type, appdef.NewQName("test", "unknown")), "should be nil if unknown")

	t.Run("should be validation error", func(t *testing.T) {
		t.Run("if no cron string", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			job := wsb.AddJob(jobName)
			_, err := adb.Build()
			require.Error(err, require.Has(job))
		})

		t.Run("if invalid cron string", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			job := wsb.AddJob(jobName)
			job.SetCronSchedule("naked 🔫")
			_, err := adb.Build()
			require.Error(err, require.Has(job), require.Has("naked 🔫"))
		})
	})
}
