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
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Projectors(t *testing.T) {
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
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddCRecord(recName).SetComment("record 1 is trigger for projector")
		_ = wsb.AddCRecord(rec2Name)
		wsb.AddCDoc(docName).SetComment("doc is state for projector")

		v := wsb.AddView(viewName)
		v.Key().PartKey().AddDataField("id", appdef.SysData_RecordID)
		v.Key().ClustCols().AddDataField("name", appdef.SysData_String)
		v.Value().AddDataField("data", appdef.SysData_bytes, false, constraints.MaxLen(1024))
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

	t.Run("should be ok to find builded projectors", func(t *testing.T) {

		t.Run(fmt.Sprint(prjRecName), func(t *testing.T) {
			typ := app.Type(prjRecName)
			require.Equal(appdef.TypeKind_Projector, typ.Kind())

			p, ok := typ.(appdef.IProjector)
			require.True(ok)
			require.Equal(appdef.TypeKind_Projector, p.Kind())

			prj := appdef.Projector(app.Type, prjRecName)
			require.Equal(appdef.TypeKind_Projector, prj.Kind())
			require.Equal(wsName, prj.Workspace().QName())
			require.Equal(p, prj)

			require.Equal(prjRecName.Entity(), prj.Name())
			require.Equal(appdef.ExtensionEngineKind_BuiltIn, prj.Engine())
			require.True(prj.Sync())
			require.True(prj.WantErrors())

			t.Run("should be ok enum events", func(t *testing.T) {
				cnt := 0
				for _, ev := range prj.Events() {
					cnt++
					switch cnt {
					case 1:
						require.EqualValues([]appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Activate, appdef.OperationKind_Deactivate}, ev.Ops())
						require.Equal(appdef.FilterKind_QNames, ev.Filter().Kind())
						require.EqualValues([]appdef.QName{recName}, ev.Filter().QNames())
						require.Equal("run projector every time when test.record is changed", ev.Comment())
					default:
						require.Failf("unexpected event", "event: %v", ev)
					}
				}
				require.Equal(1, cnt)
			})

			t.Run("should be ok to check Triggers", func(t *testing.T) {
				tests := []struct {
					name string
					op   appdef.OperationKind
					t    appdef.IType
					want bool
				}{
					{"ON EXECUTE test.cmd", appdef.OperationKind_Execute, app.Type(cmdName), false},
					{"ON INSERT test.rec", appdef.OperationKind_Insert, app.Type(recName), true},
				}
				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						require.Equal(tt.want, prj.Triggers(tt.op, tt.t))
					})
				}
			})
		})

		t.Run(fmt.Sprint(prjCmdName), func(t *testing.T) {
			prj := appdef.Projector(app.Type, prjCmdName)
			require.NotNil(prj)
			require.Equal(appdef.ExtensionEngineKind_WASM, prj.Engine())
			require.Equal("customExtensionName", prj.Name())

			t.Run("should be ok enum events", func(t *testing.T) {
				cnt := 0
				for _, ev := range prj.Events() {
					cnt++
					switch cnt {
					case 1:
						require.Equal([]appdef.OperationKind{appdef.OperationKind_Execute}, ev.Ops())
						require.Equal(appdef.FilterKind_QNames, ev.Filter().Kind())
						require.EqualValues([]appdef.QName{cmdName}, ev.Filter().QNames())
						require.Equal("run projector every time when test.command is executed", ev.Comment())
					case 2:
						require.Equal([]appdef.OperationKind{appdef.OperationKind_ExecuteWithParam}, ev.Ops())
						require.Equal(appdef.FilterKind_QNames, ev.Filter().Kind())
						require.EqualValues([]appdef.QName{objName}, ev.Filter().QNames())
						require.Equal("run projector every time when command with test.object is executed", ev.Comment())
					default:
						require.Failf("unexpected event", "event: %v", ev)
					}
				}
				require.Equal(2, cnt)
			})

			t.Run("should be ok to check Triggers", func(t *testing.T) {
				tests := []struct {
					name string
					op   appdef.OperationKind
					t    appdef.IType
					want bool
				}{
					{"ON EXECUTE test.cmd", appdef.OperationKind_Execute, app.Type(cmdName), true},
					{"ON INSERT test.rec", appdef.OperationKind_Insert, app.Type(recName), false},
				}
				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						require.Equal(tt.want, prj.Triggers(tt.op, tt.t))
					})
				}
			})
		})
	})

	t.Run("should be ok to enum projectors", func(t *testing.T) {
		names := appdef.QNames{}
		for p := range appdef.Projectors(app.Types()) {
			names.Add(p.QName())
		}
		require.EqualValues(appdef.QNamesFrom(prjRecName, prjCmdName), names)
	})

	require.Nil(appdef.Projector(app.Type, appdef.NewQName("test", "unknown")), "should be nil if unknown")

	t.Run("should be validation error", func(t *testing.T) {
		t.Run("if no filter matches", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")

			prj := adb.AddWorkspace(wsName).AddProjector(prjCmdName)
			prj.Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.AllWSFunctions(wsName))

			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.HasAll(prj, "no matches", wsName))
		})
	})

	t.Run("should be panics", func(t *testing.T) {

		t.Run("if invalid events", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			prj := wsb.AddProjector(prjRecName)
			require.Panics(func() {
				prj.Events().Add(
					[]appdef.OperationKind{appdef.OperationKind_Inherits}, // <-- invalid operation
					filter.True())
			}, require.Is(appdef.ErrUnsupportedError), require.HasAll("operations", "Inherits"))
			require.Panics(func() {
				prj.Events().Add([]appdef.OperationKind{appdef.OperationKind_Execute},
					nil) // <-- missed filter
			}, require.Is(appdef.ErrMissedError), require.Has("filter"))
		})
	})
}
