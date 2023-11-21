/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddProjector(t *testing.T) {
	require := require.New(t)

	var app IAppDef
	sysRecords, sysViews, sysWLog := NewQName(SysPackage, "records"), NewQName(SysPackage, "views"), NewQName(SysPackage, "WLog")
	prjName, recName, docName, viewName := NewQName("test", "projector"), NewQName("test", "record"), NewQName("test", "document"), NewQName("test", "view")
	cmdName := NewQName("test", "command")

	t.Run("must be ok to add projector", func(t *testing.T) {
		appDef := New()

		appDef.AddCRecord(recName).SetComment("record is trigger for projector")
		appDef.AddCDoc(docName).SetComment("doc is state for projector")

		v := appDef.AddView(viewName)
		v.KeyBuilder().PartKeyBuilder().AddDataField("id", SysData_RecordID)
		v.KeyBuilder().ClustColsBuilder().AddDataField("name", SysData_String)
		v.ValueBuilder().AddDataField("data", SysData_bytes, false, MaxLen(1024))
		v.SetComment("view is intent for projector")

		_ = appDef.AddCommand(cmdName)

		prj := appDef.AddProjector(prjName)

		t.Run("test newly created projector", func(t *testing.T) {
			require.Equal(TypeKind_Projector, prj.Kind())
			require.Equal(prjName, prj.QName())
			require.False(prj.Sync())
		})

		prj.
			SetSync(true).
			AddEvent(recName, ProjectorEventKind_AnyChanges...).
			SetEventComment(recName, fmt.Sprintf("run projector after change %v", recName)).
			AddEvent(cmdName).
			SetEventComment(cmdName, fmt.Sprintf("run projector after execute %v", cmdName)).
			AddState(sysRecords, docName, recName).AddState(sysWLog).
			AddIntent(sysViews, viewName)

		t.Run("must be ok to build", func(t *testing.T) {
			a, err := appDef.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	t.Run("must be ok to find builded projector", func(t *testing.T) {
		typ := app.Type(prjName)
		require.Equal(TypeKind_Projector, typ.Kind())

		p, ok := typ.(IProjector)
		require.True(ok)
		require.Equal(TypeKind_Projector, p.Kind())

		prj := app.Projector(prjName)
		require.Equal(TypeKind_Projector, prj.Kind())
		require.Equal(p, prj)

		require.Equal(prjName.Entity(), prj.Name())
		require.Equal(ExtensionEngineKind_BuiltIn, prj.Engine())

		t.Run("must be ok enum events", func(t *testing.T) {
			cnt := 0
			prj.Events(func(e IProjectorEvent) {
				cnt++
				switch cnt {
				case 1:
					require.Equal(cmdName, e.On().QName())
					require.EqualValues([]ProjectorEventKind{ProjectorEventKind_Execute}, e.Kind())
					require.Contains(e.Comment(), "run projector after execute")
					require.Contains(e.Comment(), cmdName.String())
				case 2:
					require.Equal(recName, e.On().QName())
					require.EqualValues(ProjectorEventKind_AnyChanges, e.Kind())
					require.Contains(e.Comment(), "run projector after change")
					require.Contains(e.Comment(), recName.String())
				default:
					require.Failf("unexpected event", "event: %v", e)
				}
			})
			require.Equal(2, cnt)
		})

		t.Run("must be ok enum states", func(t *testing.T) {
			cnt := 0
			prj.States(func(s QName, names QNames) {
				cnt++
				switch cnt {
				case 1:
					require.Equal(sysWLog, s)
					require.Empty(names)
				case 2:
					require.Equal(sysRecords, s)
					require.EqualValues(QNames{docName, recName}, names)
				default:
					require.Failf("unexpected state", "state: %v, names: %v", s, names)
				}
			})
			require.Equal(2, cnt)
		})

		t.Run("must be ok enum intents", func(t *testing.T) {
			cnt := 0
			prj.Intents(func(s QName, names QNames) {
				cnt++
				switch cnt {
				case 1:
					require.Equal(sysViews, s)
					require.EqualValues(QNames{viewName}, names)
				default:
					require.Failf("unexpected intent", "intent: %v, names: %v", s, names)
				}
			})
			require.Equal(1, cnt)
		})
	})

	t.Run("must be ok to enum projectors", func(t *testing.T) {
		cnt := 0
		app.Projectors(func(p IProjector) {
			cnt++
			switch cnt {
			case 1:
				require.Equal(TypeKind_Projector, p.Kind())
				require.Equal(prjName, p.QName())
			default:
				require.Failf("unexpected projector", "projector: %v", p)
			}
		})
		require.Equal(1, cnt)
	})

	t.Run("check nil returns", func(t *testing.T) {
		require.Nil(app.Projector(NewQName("test", "unknown")))
	})

	t.Run("more add project checks", func(t *testing.T) {
		apb := New()
		_ = apb.AddCRecord(recName)
		prj := apb.AddProjector(prjName)
		prj.
			SetEngine(ExtensionEngineKind_WASM).
			SetName("customExtensionName")
		prj.
			AddEvent(recName, ProjectorEventKind_Insert, ProjectorEventKind_Update).
			AddEvent(recName, ProjectorEventKind_Activate, ProjectorEventKind_Deactivate).
			SetEventComment(recName, "event can be added twice")
		app, err := apb.Build()
		require.NoError(err)

		p := app.Projector(prjName)

		require.Equal("customExtensionName", p.Name())
		require.Equal(ExtensionEngineKind_WASM, p.Engine())

		t.Run("must be ok enum events", func(t *testing.T) {
			cnt := 0
			p.Events(func(e IProjectorEvent) {
				cnt++
				switch cnt {
				case 1:
					require.Equal(recName, e.On().QName())
					require.EqualValues(ProjectorEventKind_AnyChanges, e.Kind())
					require.Equal("event can be added twice", e.Comment())
				default:
					require.Failf("unexpected event", "event: %v", e)
				}
			})
			require.Equal(1, cnt)
		})
	})

	t.Run("panic if name is empty", func(t *testing.T) {
		apb := New()
		require.Panics(func() {
			apb.AddProjector(NullQName)
		})
	})

	t.Run("panic if name is invalid", func(t *testing.T) {
		apb := New()
		require.Panics(func() {
			apb.AddProjector(NewQName("naked", "🔫"))
		})
	})

	t.Run("panic if type with name already exists", func(t *testing.T) {
		testName := NewQName("test", "dupe")
		apb := New()
		apb.AddObject(testName)
		require.Panics(func() {
			apb.AddProjector(testName)
		})
	})

	t.Run("panic if extension name is invalid", func(t *testing.T) {
		apb := New()
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() {
			prj.SetName("naked 🔫")
		})
	})

	t.Run("panic if event type is empty", func(t *testing.T) {
		apb := New()
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() {
			prj.AddEvent(NullQName, ProjectorEventKind_AnyChanges...)
		})
	})

	t.Run("panic if event type is unknown", func(t *testing.T) {
		apb := New()
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() {
			prj.AddEvent(NewQName("test", "unknown"), ProjectorEventKind_AnyChanges...)
		})
	})

	t.Run("panic if event type is not record or command", func(t *testing.T) {
		apb := New()
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() {
			prj.AddEvent(QNameANY, ProjectorEventKind_Execute)
		})
	})

	t.Run("panic if event is incompatible with type", func(t *testing.T) {
		apb := New()
		_ = apb.AddCRecord(recName)
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() {
			prj.AddEvent(recName, ProjectorEventKind_Execute)
		})
	})

	t.Run("panic if comment unknown event", func(t *testing.T) {
		apb := New()
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() {
			prj.SetEventComment(NewQName("test", "unknown"), "comment for unknown event should be panic")
		})
	})
}
