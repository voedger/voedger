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

	t.Run("must be ok to add projector", func(t *testing.T) {
		appDef := New()

		appDef.AddCRecord(recName).SetComment("record is trigger for projector")
		appDef.AddCDoc(docName).SetComment("doc is state for projector")

		v := appDef.AddView(viewName)
		v.KeyBuilder().PartKeyBuilder().AddDataField("id", SysData_RecordID)
		v.KeyBuilder().ClustColsBuilder().AddDataField("name", SysData_String)
		v.ValueBuilder().AddDataField("data", SysData_bytes, false, MaxLen(1024))
		v.SetComment("view is intent for projector")

		prj := appDef.AddProjector(prjName)

		t.Run("test newly created projector", func(t *testing.T) {
			require.Equal(TypeKind_Projector, prj.Kind())
			require.Equal(prjName, prj.QName())
			require.False(prj.Sync())
		})

		prj.
			SetSync(true).
			SetExtension("", ExtensionEngineKind_BuiltIn, "projector code comment").
			AddEvent(recName, ProjectorEventKind_Any...).
			SetEventComment(recName, fmt.Sprintf("run projector every time when %v is changed", recName)).
			AddEvent(QNameANY, ProjectorEventKind_Deactivate).
			AddEvent(QNameANY, ProjectorEventKind_Activate). // it is ok to add kinds for same record
			SetEventComment(recName, fmt.Sprintf("run projector every time when %v is changed", recName)).
			AddState(sysRecords, docName, recName).AddState(sysWLog).
			AddIntent(sysViews, viewName)

		// and it is ok to add event for any record

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

		require.Equal(prjName.Entity(), prj.Extension().Name())
		require.Equal(ExtensionEngineKind_BuiltIn, prj.Extension().Engine())
		require.Equal("projector code comment", prj.Extension().Comment())

		t.Run("must be ok enum events", func(t *testing.T) {
			cnt := 0
			prj.Events(func(e IProjectorEvent) {
				cnt++
				switch cnt {
				case 1:
					require.Equal(AnyType, e.On())
					require.EqualValues([]ProjectorEventKind{ProjectorEventKind_Activate, ProjectorEventKind_Deactivate}, e.Kind())
				case 2:
					require.Equal(recName, e.On().QName())
					require.Equal(TypeKind_CRecord, e.On().Kind())
					require.EqualValues(ProjectorEventKind_Any, e.Kind())
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
			prj.SetExtension("naked 🔫", ExtensionEngineKind_BuiltIn)
		})
		require.NotPanics(func() {
			prj.SetExtension("customName", ExtensionEngineKind_BuiltIn)
		}, "but no panic if name is valid identifier")
	})

	t.Run("panic if event type is empty", func(t *testing.T) {
		apb := New()
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() {
			prj.AddEvent(NullQName, ProjectorEventKind_Any...)
		})
	})

	t.Run("panic if event type is unknown", func(t *testing.T) {
		apb := New()
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() {
			prj.AddEvent(NewQName("test", "unknown"), ProjectorEventKind_Any...)
		})
		require.NotPanics(func() {
			prj.AddEvent(QNameANY, ProjectorEventKind_Any...)
		}, "but ok if event is any record")
	})

	t.Run("panic if comment unknown event", func(t *testing.T) {
		apb := New()
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() {
			prj.SetEventComment(NewQName("test", "unknown"), "comment for unknown event should be panic")
		})
	})
}
