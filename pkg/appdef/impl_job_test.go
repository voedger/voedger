/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddJob(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	wsName := NewQName("test", "workspace")

	sysViews := NewQName(SysPackage, "views")
	viewName := NewQName("test", "view")
	cronSchedule := `@every 2m30s`
	jobName := NewQName("test", "job")

	t.Run("should be ok to add job", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		v := wsb.AddView(viewName)
		v.Key().PartKey().AddDataField("id", SysData_RecordID)
		v.Key().ClustCols().AddDataField("name", SysData_String)
		v.Value().AddDataField("data", SysData_bytes, false, MaxLen(1024))
		v.SetComment("view is state for job")

		job := wsb.AddJob(jobName)

		job.
			SetCronSchedule(cronSchedule).
			States().Add(sysViews, viewName).SetComment(sysViews, "view is state for job")

		t.Run("should be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	testWithJobs := func(tested IWithJobs) {

		t.Run("should be ok to find builded job", func(t *testing.T) {
			typ := tested.(IWithTypes).Type(jobName)
			require.Equal(TypeKind_Job, typ.Kind())

			j, ok := typ.(IJob)
			require.True(ok)
			require.Equal(TypeKind_Job, j.Kind())

			job := tested.Job(jobName)
			require.Equal(TypeKind_Job, job.Kind())
			require.Equal(wsName, job.Workspace().QName())
			require.Equal(j, job)

			require.Equal(jobName.Entity(), job.Name())
			require.Equal(ExtensionEngineKind_BuiltIn, job.Engine())

			require.Equal(cronSchedule, job.CronSchedule())

			t.Run("should be ok enum states", func(t *testing.T) {
				cnt := 0
				for s := range job.States().Enum {
					cnt++
					switch cnt {
					case 1:
						require.Equal(sysViews, s.Name())
						require.EqualValues(QNames{viewName}, s.Names())
					default:
						require.Failf("unexpected state", "state: %v", s)
					}
				}
				require.Equal(1, cnt)
				require.Equal(cnt, job.States().Len())

				t.Run("should be ok to get states as map", func(t *testing.T) {
					states := job.States().Map()
					require.Len(states, 1)
					require.Contains(states, sysViews)
					require.EqualValues(QNames{viewName}, states[sysViews])
				})

				t.Run("should be ok to get state by name", func(t *testing.T) {
					state := job.States().Storage(sysViews)
					require.NotNil(state)
					require.Equal(sysViews, state.Name())
					require.EqualValues(QNames{viewName}, state.Names())

					require.Nil(job.States().Storage(NewQName("test", "unknown")), "should be nil for unknown state")
				})
			})
		})

		t.Run("should be ok to enum jobs", func(t *testing.T) {
			cnt := 0
			for j := range tested.Jobs {
				cnt++
				switch cnt {
				case 1:
					require.Equal(TypeKind_Job, j.Kind())
					require.Equal(jobName, j.QName())
				default:
					require.Failf("unexpected job", "job: %v", j)
				}
			}
			require.Equal(1, cnt)
		})

		require.Nil(tested.Job(NewQName("test", "unknown")), "should be nil if unknown")
	}

	testWithJobs(app)
	testWithJobs(app.Workspace(wsName))

	t.Run("more add job checks", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(wsName)

		job := wsb.AddJob(jobName)
		job.
			SetEngine(ExtensionEngineKind_WASM).
			SetName("customExtensionName")
		job.SetCronSchedule(cronSchedule)
		app, err := adb.Build()
		require.NoError(err)

		j := app.Job(jobName)

		require.Equal("customExtensionName", j.Name())
		require.Equal(ExtensionEngineKind_WASM, j.Engine())
		require.Equal(cronSchedule, j.CronSchedule())
	})

	t.Run("should be validation error", func(t *testing.T) {
		t.Run("if unknown names in states", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			job := wsb.AddJob(jobName)
			job.States().
				Add(sysViews, viewName, NewQName("test", "unknown"))
			_, err := adb.Build()
			require.Error(err, require.Is(ErrNotFoundError), require.Has("test.unknown"))
		})

		t.Run("if no cron string", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			job := wsb.AddJob(jobName)
			_, err := adb.Build()
			require.Error(err, require.Has(job))
		})

		t.Run("if invalid cron string", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			job := wsb.AddJob(jobName)
			job.SetCronSchedule("naked 🔫")
			_, err := adb.Build()
			require.Error(err, require.Has(job), require.Has("naked 🔫"))
		})

		t.Run("if wrong intents", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)

			v := wsb.AddView(viewName)
			v.Key().PartKey().AddDataField("id", SysData_RecordID)
			v.Key().ClustCols().AddDataField("name", SysData_String)
			v.Value().AddDataField("data", SysData_bytes, false, MaxLen(1024))

			job := wsb.AddJob(jobName)
			job.SetCronSchedule("@hourly")
			job.Intents().
				Add(sysViews, viewName).SetComment(sysViews, "error here: job shall not have intents")

			_, err := adb.Build()
			require.Error(err, require.Is(ErrUnsupportedError), require.Has(job))
		})
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if invalid name", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)

			require.Panics(func() { wsb.AddJob(NullQName) },
				require.Is(ErrMissedError))
			require.Panics(func() { wsb.AddJob(NewQName("naked", "🔫")) },
				require.Is(ErrInvalidError), require.Has("naked.🔫"))
		})

		t.Run("if type with name already exists", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			testName := NewQName("test", "dupe")
			wsb.AddObject(testName)

			require.Panics(func() { wsb.AddJob(testName) },
				require.Is(ErrAlreadyExistsError), require.Has(testName))
		})

		t.Run("if extension name is invalid", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			job := wsb.AddJob(jobName)

			require.Panics(func() { job.SetName("naked 🔫") },
				require.Is(ErrInvalidError), require.Has("naked 🔫"))
		})

		t.Run("if invalid states", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			job := wsb.AddJob(jobName)

			require.Panics(func() { job.States().Add(NullQName) },
				require.Is(ErrMissedError))
			require.Panics(func() { job.States().Add(NewQName("naked", "🔫")) },
				require.Is(ErrInvalidError), require.Has("naked.🔫"))
			require.Panics(func() { job.States().Add(sysViews, NewQName("naked", "🔫")) },
				require.Is(ErrInvalidError), require.Has("🔫"))
			require.Panics(func() { job.States().SetComment(NewQName("unknown", "storage"), "comment") },
				require.Is(ErrNotFoundError), require.Has("unknown.storage"))
		})
	})
}
