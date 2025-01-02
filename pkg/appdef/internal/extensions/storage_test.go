/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Storages(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")

	// storages
	sysRecords, sysViews, sysWLog := appdef.NewQName(appdef.SysPackage, "records"), appdef.NewQName(appdef.SysPackage, "views"), appdef.NewQName(appdef.SysPackage, "WLog")

	// state and intent
	docName, viewName := appdef.NewQName("test", "document"), appdef.NewQName("test", "view")

	prjName := appdef.NewQName("test", "projector")

	t.Run("test storages via projector states and intents", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddCDoc(docName)

		v := wsb.AddView(viewName)
		v.Key().PartKey().AddDataField("id", appdef.SysData_RecordID)
		v.Key().ClustCols().AddDataField("name", appdef.SysData_String)
		v.Value().AddDataField("data", appdef.SysData_bytes, false)

		prj := wsb.AddProjector(prjName)
		prj.Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Activate, appdef.OperationKind_Deactivate},
			filter.QNames(docName))
		prj.States().
			Add(sysRecords, docName).
			Add(sysWLog)
		prj.Intents().
			Add(sysViews, viewName).SetComment(sysViews, "view is intent for projector")

		t.Run("should be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	t.Run("should be ok to find builded projectors", func(t *testing.T) {

		t.Run(fmt.Sprint(prjName), func(t *testing.T) {
			prj := appdef.Projector(app.Type, prjName)
			require.NotNil(prj)

			t.Run("should be ok enum states", func(t *testing.T) {
				cnt := 0
				for s := range prj.States().Enum {
					cnt++
					switch cnt {
					case 1: // "sys.WLog" < "sys.records" (`W` < `r`)
						require.Equal(sysWLog, s.Name())
						require.Empty(s.Names())
						require.Equal(`Storage «sys.WLog» []`, fmt.Sprint(s))
					case 2:
						require.Equal(sysRecords, s.Name())
						require.EqualValues(appdef.QNames{docName}, s.Names())
						require.Equal(`Storage «sys.records» [test.document]`, fmt.Sprint(s))
					default:
						require.Failf("unexpected state", "state: %v", s)
					}
				}
				require.Equal(2, cnt)
				require.Equal(cnt, prj.States().Len())

				t.Run("should be ok to break enum states", func(t *testing.T) {
					cnt := 0
					for range prj.States().Enum {
						cnt++
						break
					}
					require.Equal(1, cnt)
				})

				t.Run("should be ok to get states as map", func(t *testing.T) {
					states := prj.States().Map()
					require.Len(states, 2)
					require.Contains(states, sysRecords)
					require.EqualValues(appdef.QNames{docName}, states[sysRecords])
					require.Contains(states, sysWLog)
					require.Empty(states[sysWLog])
				})

				t.Run("should be ok to get state by name", func(t *testing.T) {
					state := prj.States().Storage(sysRecords)
					require.NotNil(state)
					require.Equal(sysRecords, state.Name())
					require.EqualValues(appdef.QNames{docName}, state.Names())

					require.Nil(prj.States().Storage(appdef.NewQName("test", "unknown")), "should be nil for unknown state")
				})
			})

			t.Run("should be ok enum intents", func(t *testing.T) {
				cnt := 0
				for i := range prj.Intents().Enum {
					cnt++
					switch cnt {
					case 1:
						require.Equal(sysViews, i.Name())
						require.EqualValues(appdef.QNames{viewName}, i.Names())
						require.Equal("view is intent for projector", i.Comment())
						require.Equal(`Storage «sys.views» [test.view]`, fmt.Sprint(i))
					default:
						require.Failf("unexpected intent", "intent: %v", i)
					}
				}
				require.Equal(1, cnt)
				require.Equal(cnt, prj.Intents().Len())

				t.Run("should be ok to break enum intents", func(t *testing.T) {
					cnt := 0
					for range prj.Intents().Enum {
						cnt++
						break
					}
					require.Equal(1, cnt)
				})

				t.Run("should be ok to get intents as map", func(t *testing.T) {
					intents := prj.Intents().Map()
					require.Len(intents, 1)
					require.Contains(intents, sysViews)
					require.EqualValues(appdef.QNames{viewName}, intents[sysViews])
				})

				t.Run("should be ok to get intent by name", func(t *testing.T) {
					intent := prj.Intents().Storage(sysViews)
					require.NotNil(intent)
					require.Equal(sysViews, intent.Name())
					require.EqualValues(appdef.QNames{viewName}, intent.Names())

					require.Nil(prj.Intents().Storage(appdef.NewQName("test", "unknown")), "should be nil for unknown intent")
				})
			})
		})
	})

	t.Run("should be validation error", func(t *testing.T) {
		t.Run("if unknown names in states", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")

			wsb := adb.AddWorkspace(wsName)

			wsb.AddCDoc(docName)
			prj := wsb.AddProjector(prjName)
			prj.Events().Add([]appdef.OperationKind{appdef.OperationKind_Insert}, filter.QNames(docName))
			prj.States().
				Add(appdef.NewQName("sys", "records"), docName, appdef.NewQName("test", "unknown"))
			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.Has("test.unknown"))
		})
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if invalid states", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			prj := wsb.AddProjector(prjName)
			require.Panics(func() { prj.States().Add(appdef.NullQName) },
				require.Is(appdef.ErrMissedError))
			require.Panics(func() { prj.States().Add(appdef.NewQName("appdef.naked", "🔫")) },
				require.Is(appdef.ErrInvalidError), require.Has("naked.🔫"))
			require.Panics(func() { prj.States().Add(sysRecords, appdef.NewQName("naked", "🔫")) },
				require.Is(appdef.ErrInvalidError), require.Has("🔫"))
			require.Panics(func() { prj.States().SetComment(appdef.NewQName("unknown", "storage"), "comment") },
				require.Is(appdef.ErrNotFoundError), require.Has("unknown.storage"))
		})

		t.Run("if invalid intents", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			prj := wsb.AddProjector(prjName)
			require.Panics(func() { prj.Intents().Add(appdef.NullQName) },
				require.Is(appdef.ErrMissedError))
			require.Panics(func() { prj.Intents().Add(appdef.NewQName("appdef.naked", "🔫")) },
				require.Is(appdef.ErrInvalidError), require.Has("naked.🔫"))
			require.Panics(func() { prj.Intents().Add(sysRecords, appdef.NewQName("naked", "🔫")) },
				require.Is(appdef.ErrInvalidError), require.Has("🔫"))
			require.Panics(func() { prj.Intents().SetComment(appdef.NewQName("unknown", "storage"), "comment") },
				require.Is(appdef.ErrNotFoundError), require.Has("unknown.storage"))
		})
	})
}