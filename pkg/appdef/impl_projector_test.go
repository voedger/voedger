/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddProjector(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")

	// storages
	sysRecords, sysViews, sysWLog := appdef.NewQName(appdef.SysPackage, "records"), appdef.NewQName(appdef.SysPackage, "views"), appdef.NewQName(appdef.SysPackage, "WLog")

	// state and intent
	docName, viewName := appdef.NewQName("test", "document"), appdef.NewQName("test", "view")

	// records to trigger
	recName, rec2Name := appdef.NewQName("test", "record"), appdef.NewQName("test", "record2")

	// command with params to trigger
	cmdName, objName := appdef.NewQName("test", "command"), appdef.NewQName("test", "object")

	prjRecName := appdef.NewQName("test", "recProjector")
	prjCmdName := appdef.NewQName("test", "cmdProjector")

	t.Run("should be ok to add projector", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddCRecord(recName).SetComment("record 1 is trigger for projector")
		_ = wsb.AddCRecord(rec2Name)
		wsb.AddCDoc(docName).SetComment("doc is state for projector")

		v := wsb.AddView(viewName)
		v.Key().PartKey().AddDataField("id", appdef.SysData_RecordID)
		v.Key().ClustCols().AddDataField("name", appdef.SysData_String)
		v.Value().AddDataField("data", appdef.SysData_bytes, false, appdef.MaxLen(1024))
		v.SetComment("view is intent for projector")

		_ = wsb.AddObject(objName)
		wsb.AddCommand(cmdName).SetParam(objName)

		prjRec := wsb.AddProjector(prjRecName)
		prjRec.Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Activate, appdef.OperationKind_Deactivate},
			filter.QNames(recName),
			fmt.Sprintf("run projector every time when %v is changed", recName))
		prjRec.
			SetSync(true).
			SetWantErrors()
		prjRec.States().
			Add(sysRecords, docName).
			Add(sysRecords, recName, rec2Name). // should be ok to add storage «sys.records» twice, qnames must concate
			Add(sysWLog)
		prjRec.Intents().
			Add(sysViews, viewName).SetComment(sysViews, "view is intent for projector")

		prjCmd := wsb.AddProjector(prjCmdName)
		prjCmd.Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.QNames(cmdName),
			fmt.Sprintf("run projector every time when %v is executed", cmdName))
		prjCmd.Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_ExecuteWithParam},
			filter.QNames(objName),
			fmt.Sprintf("run projector every time when command with %v is executed", objName))
		prjCmd.SetEngine(appdef.ExtensionEngineKind_WASM)
		prjCmd.SetName("customExtensionName")
		prjCmd.Intents().Add(sysViews, viewName)

		t.Run("should be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	testWith := func(tested testedTypes) {

		t.Run("should be ok to find builded projectors", func(t *testing.T) {

			t.Run(fmt.Sprint(prjRecName), func(t *testing.T) {
				typ := tested.Type(prjRecName)
				require.Equal(appdef.TypeKind_Projector, typ.Kind())

				p, ok := typ.(appdef.IProjector)
				require.True(ok)
				require.Equal(appdef.TypeKind_Projector, p.Kind())

				prj := appdef.Projector(tested.Type, prjRecName)
				require.Equal(appdef.TypeKind_Projector, prj.Kind())
				require.Equal(wsName, prj.Workspace().QName())
				require.Equal(p, prj)

				require.Equal(prjRecName.Entity(), prj.Name())
				require.Equal(appdef.ExtensionEngineKind_BuiltIn, prj.Engine())
				require.True(prj.Sync())
				require.True(prj.WantErrors())

				t.Run("should be ok enum events", func(t *testing.T) {
					cnt := 0
					for ev := range prj.Events() {
						cnt++
						switch cnt {
						case 1:
							require.EqualValues([]appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Activate, appdef.OperationKind_Deactivate}, slices.Collect(ev.Ops()))
							require.Equal(appdef.FilterKind_QNames, ev.Filter().Kind())
							require.EqualValues([]appdef.QName{recName}, slices.Collect(ev.Filter().QNames()))
							require.Equal("run projector every time when test.record is changed", ev.Comment())
						default:
							require.Failf("unexpected event", "event: %v", ev)
						}
					}
					require.Equal(1, cnt)
				})

				t.Run("should be ok enum states", func(t *testing.T) {
					cnt := 0
					for s := range prj.States().Enum {
						cnt++
						switch cnt {
						case 1: // "sys.WLog" < "sys.records" (`W` < `r`)
							require.Equal(sysWLog, s.Name())
							require.Empty(s.Names())
						case 2:
							require.Equal(sysRecords, s.Name())
							require.EqualValues(appdef.QNames{docName, recName, rec2Name}, s.Names())
						default:
							require.Failf("unexpected state", "state: %v", s)
						}
					}
					require.Equal(2, cnt)
					require.Equal(cnt, prj.States().Len())

					t.Run("should be ok to get states as map", func(t *testing.T) {
						states := prj.States().Map()
						require.Len(states, 2)
						require.Contains(states, sysRecords)
						require.EqualValues(appdef.QNames{docName, recName, rec2Name}, states[sysRecords])
						require.Contains(states, sysWLog)
						require.Empty(states[sysWLog])
					})

					t.Run("should be ok to get state by name", func(t *testing.T) {
						state := prj.States().Storage(sysRecords)
						require.NotNil(state)
						require.Equal(sysRecords, state.Name())
						require.EqualValues(appdef.QNames{docName, recName, rec2Name}, state.Names())

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
						default:
							require.Failf("unexpected intent", "intent: %v", i)
						}
					}
					require.Equal(1, cnt)
					require.Equal(cnt, prj.Intents().Len())

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

			t.Run(fmt.Sprint(prjCmdName), func(t *testing.T) {
				prj := appdef.Projector(tested.Type, prjCmdName)
				require.NotNil(prj)
				require.Equal(appdef.ExtensionEngineKind_WASM, prj.Engine())
				require.Equal("customExtensionName", prj.Name())

				t.Run("should be ok enum events", func(t *testing.T) {
					cnt := 0
					for ev := range prj.Events() {
						cnt++
						switch cnt {
						case 1:
							require.Equal([]appdef.OperationKind{appdef.OperationKind_Execute}, slices.Collect(ev.Ops()))
							require.Equal(appdef.FilterKind_QNames, ev.Filter().Kind())
							require.EqualValues([]appdef.QName{cmdName}, slices.Collect(ev.Filter().QNames()))
							require.Equal("run projector every time when test.command is executed", ev.Comment())
						case 2:
							require.Equal([]appdef.OperationKind{appdef.OperationKind_ExecuteWithParam}, slices.Collect(ev.Ops()))
							require.Equal(appdef.FilterKind_QNames, ev.Filter().Kind())
							require.EqualValues([]appdef.QName{objName}, slices.Collect(ev.Filter().QNames()))
							require.Equal("run projector every time when command with test.object is executed", ev.Comment())
						default:
							require.Failf("unexpected event", "event: %v", ev)
						}
					}
					require.Equal(2, cnt)
				})

				require.Empty(prj.States().Map())
				require.EqualValues(map[appdef.QName]appdef.QNames{sysViews: {viewName}}, prj.Intents().Map())
			})
		})

		t.Run("should be ok to enum projectors", func(t *testing.T) {
			names := appdef.QNames{}
			for p := range appdef.Projectors(tested.Types()) {
				names.Add(p.QName())
			}
			require.EqualValues(appdef.QNamesFrom(prjRecName, prjCmdName), names)
		})

		require.Nil(appdef.Projector(tested.Type, appdef.NewQName("test", "unknown")), "should be nil if unknown")
	}

	testWith(app)
	testWith(app.Workspace(wsName))

	t.Run("should be validation error", func(t *testing.T) {
		t.Run("if no filter matches", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")

			prj := adb.AddWorkspace(wsName).AddProjector(prjCmdName)
			prj.Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.AllFunctions(wsName))

			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.HasAll(prj, "no matches", wsName))
		})

		t.Run("if unknown names in states", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")

			wsb := adb.AddWorkspace(wsName)

			wsb.AddCRecord(recName)
			prj := wsb.AddProjector(prjRecName)
			prj.Events().Add([]appdef.OperationKind{appdef.OperationKind_Insert}, filter.QNames(recName))
			prj.States().
				Add(appdef.NewQName("sys", "records"), recName, appdef.NewQName("test", "unknown"))
			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.Has("test.unknown"))
		})
	})

	t.Run("should be panics", func(t *testing.T) {
		t.Run("if invalid name", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			require.Panics(func() {
				wsb.AddProjector(
					appdef.NullQName, // <-- missed name
				)
			}, require.Is(appdef.ErrMissedError))
			require.Panics(func() {
				wsb.AddProjector(
					appdef.NewQName("naked", "🔫"), // <-- invalid name
				)
			}, require.Is(appdef.ErrInvalidError), require.Has("naked.🔫"))

			testName := appdef.NewQName("test", "dupe")
			wsb.AddObject(testName)
			require.Panics(func() {
				wsb.AddProjector(
					testName, // <-- dupe name
				)
			}, require.Is(appdef.ErrAlreadyExistsError), require.Has(testName))
		})

		t.Run("if extension name is invalid", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			prj := wsb.AddProjector(prjRecName)
			require.Panics(func() { prj.SetName("naked 🔫") },
				require.Is(appdef.ErrInvalidError), require.Has("naked 🔫"))
		})

		t.Run("if invalid states", func(t *testing.T) {
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			prj := wsb.AddProjector(prjRecName)
			prj.Events().Add([]appdef.OperationKind{appdef.OperationKind_Insert}, filter.QNames(recName))
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
			adb := appdef.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			prj := wsb.AddProjector(prjRecName)
			prj.Events().Add([]appdef.OperationKind{appdef.OperationKind_Insert}, filter.QNames(recName))
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
