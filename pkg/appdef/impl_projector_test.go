/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddProjector(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	// storages
	sysRecords, sysViews, sysWLog := NewQName(SysPackage, "records"), NewQName(SysPackage, "views"), NewQName(SysPackage, "WLog")

	// state and intent
	docName, viewName := NewQName("test", "document"), NewQName("test", "view")

	// records to trigger
	recName, rec2Name := NewQName("test", "record"), NewQName("test", "record2")

	// command with params to trigger
	cmdName, objName := NewQName("test", "command"), NewQName("test", "object")

	prjName := NewQName("test", "projector")

	t.Run("must be ok to add projector", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		adb.AddCRecord(recName).SetComment("record 1 is trigger for projector")
		_ = adb.AddCRecord(rec2Name)
		adb.AddCDoc(docName).SetComment("doc is state for projector")

		v := adb.AddView(viewName)
		v.Key().PartKey().AddDataField("id", SysData_RecordID)
		v.Key().ClustCols().AddDataField("name", SysData_String)
		v.Value().AddDataField("data", SysData_bytes, false, MaxLen(1024))
		v.SetComment("view is intent for projector")

		_ = adb.AddObject(objName)
		adb.AddCommand(cmdName).SetParam(objName)

		prj := adb.AddProjector(prjName)

		prj.
			SetSync(true).
			SetWantErrors()
		prj.Events().
			Add(recName).SetComment(recName, fmt.Sprintf("run projector after change %v", recName)).
			Add(cmdName).SetComment(cmdName, fmt.Sprintf("run projector after execute %v", cmdName)).
			Add(objName).SetComment(objName, fmt.Sprintf("run projector after execute any command with parameter %v", objName))
		prj.States().
			Add(sysRecords, docName).
			Add(sysRecords, recName, rec2Name). // should be ok to add storage «sys.records» twice, qnames must concate
			Add(sysWLog)
		prj.Intents().
			Add(sysViews, viewName).SetComment(sysViews, "view is intent for projector")

		t.Run("must be ok to build", func(t *testing.T) {
			a, err := adb.Build()
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
		require.True(prj.Sync())

		t.Run("must be ok enum events", func(t *testing.T) {
			require.EqualValues(3, prj.Events().Len())

			cnt := 0
			prj.Events().Enum(func(e IProjectorEvent) {
				cnt++
				switch cnt {
				case 1:
					require.Equal(cmdName, e.On().QName())
					require.EqualValues([]ProjectorEventKind{ProjectorEventKind_Execute}, e.Kind())
					require.Contains(e.Comment(), "after execute")
					require.Contains(e.Comment(), cmdName.String())
				case 2:
					require.Equal(objName, e.On().QName())
					require.EqualValues([]ProjectorEventKind{ProjectorEventKind_ExecuteWithParam}, e.Kind())
					require.Contains(e.Comment(), "with parameter")
					require.Contains(e.Comment(), objName.String())
				case 3:
					require.Equal(recName, e.On().QName())
					require.EqualValues(ProjectorEventKind_AnyChanges, e.Kind())
					require.Contains(e.Comment(), "after change")
					require.Contains(e.Comment(), recName.String())
				default:
					require.Failf("unexpected event", "event: %v", e)
				}
			})
			require.Equal(3, cnt)

			t.Run("must be ok obtain events map", func(t *testing.T) {
				events := prj.Events().Map()
				require.Len(events, 3)
				require.Contains(events, cmdName)
				require.EqualValues([]ProjectorEventKind{ProjectorEventKind_Execute}, events[cmdName])
				require.Contains(events, objName)
				require.EqualValues([]ProjectorEventKind{ProjectorEventKind_ExecuteWithParam}, events[objName])
				require.Contains(events, recName)
				require.EqualValues(ProjectorEventKind_AnyChanges, events[recName])
			})

			t.Run("must be ok to get event by name", func(t *testing.T) {
				event := prj.Events().Event(cmdName)
				require.NotNil(event)
				require.Equal(cmdName, event.On().QName())
				require.EqualValues([]ProjectorEventKind{ProjectorEventKind_Execute}, event.Kind())

				require.Nil(prj.Events().Event(NewQName("test", "unknown")), "should be nil for unknown event")
			})
		})

		require.True(prj.WantErrors())

		t.Run("must be ok enum states", func(t *testing.T) {
			cnt := 0
			for s := range prj.States().Enum {
				cnt++
				switch cnt {
				case 1: // "sys.WLog" < "sys.records" (`W` < `r`)
					require.Equal(sysWLog, s.Name())
					require.Empty(s.Names())
				case 2:
					require.Equal(sysRecords, s.Name())
					require.EqualValues(QNames{docName, recName, rec2Name}, s.Names())
				default:
					require.Failf("unexpected state", "state: %v", s)
				}
			}
			require.Equal(2, cnt)
			require.Equal(cnt, prj.States().Len())

			t.Run("must be ok to get states as map", func(t *testing.T) {
				states := prj.States().Map()
				require.Len(states, 2)
				require.Contains(states, sysRecords)
				require.EqualValues(QNames{docName, recName, rec2Name}, states[sysRecords])
				require.Contains(states, sysWLog)
				require.Empty(states[sysWLog])
			})

			t.Run("must be ok to get state by name", func(t *testing.T) {
				state := prj.States().Storage(sysRecords)
				require.NotNil(state)
				require.Equal(sysRecords, state.Name())
				require.EqualValues(QNames{docName, recName, rec2Name}, state.Names())

				require.Nil(prj.States().Storage(NewQName("test", "unknown")), "should be nil for unknown state")
			})
		})

		t.Run("must be ok enum intents", func(t *testing.T) {
			cnt := 0
			for i := range prj.Intents().Enum {
				cnt++
				switch cnt {
				case 1:
					require.Equal(sysViews, i.Name())
					require.EqualValues(QNames{viewName}, i.Names())
					require.Equal("view is intent for projector", i.Comment())
				default:
					require.Failf("unexpected intent", "intent: %v", i)
				}
			}
			require.Equal(1, cnt)
			require.Equal(cnt, prj.Intents().Len())

			t.Run("must be ok to get intents as map", func(t *testing.T) {
				intents := prj.Intents().Map()
				require.Len(intents, 1)
				require.Contains(intents, sysViews)
				require.EqualValues(QNames{viewName}, intents[sysViews])
			})

			t.Run("must be ok to get intent by name", func(t *testing.T) {
				intent := prj.Intents().Storage(sysViews)
				require.NotNil(intent)
				require.Equal(sysViews, intent.Name())
				require.EqualValues(QNames{viewName}, intent.Names())

				require.Nil(prj.Intents().Storage(NewQName("test", "unknown")), "should be nil for unknown intent")
			})
		})
	})

	t.Run("must be ok to enum projectors", func(t *testing.T) {
		cnt := 0
		for p := range app.Projectors {
			cnt++
			switch cnt {
			case 1:
				require.Equal(TypeKind_Projector, p.Kind())
				require.Equal(prjName, p.QName())
			default:
				require.Failf("unexpected projector", "projector: %v", p)
			}
		}
		require.Equal(1, cnt)
	})

	t.Run("check nil returns", func(t *testing.T) {
		require.Nil(app.Projector(NewQName("test", "unknown")))
	})

	t.Run("more add projector checks", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		_ = adb.AddCRecord(recName)
		prj := adb.AddProjector(prjName)
		prj.
			SetEngine(ExtensionEngineKind_WASM).
			SetName("customExtensionName")
		prj.Events().
			Add(recName, ProjectorEventKind_Insert, ProjectorEventKind_Update).
			Add(recName, ProjectorEventKind_Activate, ProjectorEventKind_Deactivate). // event can be added twice
			SetComment(recName, "event can be added twice")
		app, err := adb.Build()
		require.NoError(err)

		p := app.Projector(prjName)

		require.Equal("customExtensionName", p.Name())
		require.Equal(ExtensionEngineKind_WASM, p.Engine())

		t.Run("must be ok enum events", func(t *testing.T) {
			require.EqualValues(1, p.Events().Len())
			cnt := 0
			p.Events().Enum(func(e IProjectorEvent) {
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

		require.False(p.WantErrors())
	})

	t.Run("projector validation errors", func(t *testing.T) {
		t.Run("should be error if empty events", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")

			prj := adb.AddProjector(prjName)
			_, err := adb.Build()
			require.Error(err, require.Is(ErrMissedError), require.Has(prj))
		})

		t.Run("should be error if unknown names in states", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")

			adb.AddCRecord(recName)
			prj := adb.AddProjector(prjName)
			prj.SetName("customExtensionName")
			prj.Events().
				Add(recName, ProjectorEventKind_Insert)
			prj.States().
				Add(NewQName("sys", "records"), recName, NewQName("test", "unknown"))
			_, err := adb.Build()
			require.Error(err, require.Is(ErrNotFoundError), require.Has("test.unknown"))
		})
	})

	t.Run("common panics while build projector", func(t *testing.T) {
		adb := New()
		require.Panics(func() { adb.AddProjector(NullQName) },
			require.Is(ErrMissedError))
		require.Panics(func() { adb.AddProjector(NewQName("naked", "🔫")) },
			require.Is(ErrInvalidError), require.Has("naked.🔫"))

		adb.AddPackage("test", "test.com/test")
		t.Run("panic if type with name already exists", func(t *testing.T) {
			testName := NewQName("test", "dupe")
			adb.AddObject(testName)
			require.Panics(func() { adb.AddProjector(testName) },
				require.Is(ErrAlreadyExistsError), require.Has(testName))
		})

		t.Run("panic if extension name is invalid", func(t *testing.T) {
			prj := adb.AddProjector(NewQName("test", "projector"))
			require.Panics(func() { prj.SetName("naked 🔫") },
				require.Is(ErrInvalidError), require.Has("naked 🔫"))
		})
	})

	t.Run("panics while build states", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		prj := adb.AddProjector(NewQName("test", "projector"))

		require.Panics(func() { prj.States().Add(NullQName) },
			require.Is(ErrMissedError))
		require.Panics(func() { prj.States().Add(NewQName("naked", "🔫")) },
			require.Is(ErrInvalidError), require.Has("naked.🔫"))
		require.Panics(func() { prj.States().Add(sysRecords, NewQName("naked", "🔫")) },
			require.Is(ErrInvalidError), require.Has("🔫"))
		require.Panics(func() { prj.States().SetComment(NewQName("unknown", "storage"), "comment") },
			require.Is(ErrNotFoundError), require.Has("unknown.storage"))
	})

	t.Run("panics while build intents", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		prj := adb.AddProjector(NewQName("test", "projector"))

		require.Panics(func() { prj.Intents().Add(NullQName) },
			require.Is(ErrMissedError))
		require.Panics(func() { prj.Intents().Add(NewQName("naked", "🔫")) },
			require.Is(ErrInvalidError), require.Has("naked.🔫"))
		require.Panics(func() { prj.Intents().Add(sysRecords, NewQName("naked", "🔫")) },
			require.Is(ErrInvalidError), require.Has("🔫"))
		require.Panics(func() { prj.Intents().SetComment(NewQName("unknown", "storage"), "comment") },
			require.Is(ErrNotFoundError), require.Has("unknown.storage"))
	})

	t.Run("panic while build events", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		prj := adb.AddProjector(NewQName("test", "projector"))

		require.Panics(func() { prj.Events().Add(NullQName) },
			require.Is(ErrMissedError))
		require.Panics(func() { prj.Events().Add(NewQName("test", "unknown")) },
			require.Is(ErrNotFoundError), require.Has("test.unknown"))
		require.Panics(func() { prj.Events().Add(QNameANY) },
			require.Is(ErrUnsupportedError), require.Has("ANY"))
		require.Panics(func() { prj.Events().SetComment(NewQName("test", "unknown"), "comment") },
			require.Is(ErrNotFoundError), require.Has("test.unknown"))

		t.Run("panic if event is incompatible with type", func(t *testing.T) {
			_ = adb.AddCRecord(recName)
			_ = adb.AddObject(objName)
			_ = adb.AddCommand(cmdName).SetParam(objName)

			require.Panics(func() { prj.Events().Add(prjName, ProjectorEventKind_Execute) },
				require.Is(ErrIncompatibleError), require.Has("Execute"))
			require.Panics(func() { prj.Events().Add(recName, ProjectorEventKind_Execute) },
				require.Is(ErrIncompatibleError), require.Has("Execute"))
			require.Panics(func() { prj.Events().Add(objName, ProjectorEventKind_Update) },
				require.Is(ErrIncompatibleError), require.Has("Update"))
			require.Panics(func() { prj.Events().Add(cmdName, ProjectorEventKind_ExecuteWithParam) },
				require.Is(ErrIncompatibleError), require.Has("ExecuteWith"))
		})
	})
}
