/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddProjector(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	// storages
	sysRecords, sysViews, sysWLog := NewQName(SysPackage, "records"), NewQName(SysPackage, "views"), NewQName(SysPackage, "WLog")

	// state and intent
	docName, viewName := NewQName("test", "document"), NewQName("test", "view")

	// record to trigger
	recName := NewQName("test", "record")

	// command with params to trigger
	cmdName, objName := NewQName("test", "command"), NewQName("test", "object")

	prjName := NewQName("test", "projector")

	t.Run("must be ok to add projector", func(t *testing.T) {
		appDef := New()

		appDef.AddCRecord(recName).SetComment("record is trigger for projector")
		appDef.AddCDoc(docName).SetComment("doc is state for projector")

		v := appDef.AddView(viewName)
		v.KeyBuilder().PartKeyBuilder().AddDataField("id", SysData_RecordID)
		v.KeyBuilder().ClustColsBuilder().AddDataField("name", SysData_String)
		v.ValueBuilder().AddDataField("data", SysData_bytes, false, MaxLen(1024))
		v.SetComment("view is intent for projector")

		_ = appDef.AddObject(objName)
		appDef.AddCommand(cmdName).SetParam(objName)

		prj := appDef.AddProjector(prjName)

		t.Run("test newly created projector", func(t *testing.T) {
			require.Equal(TypeKind_Projector, prj.Kind())
			require.Equal(prjName, prj.QName())
			require.False(prj.Sync())
			require.Empty(prj.EventsMap())
		})

		prj.
			SetSync(true).
			AddEvent(recName).
			SetEventComment(recName, fmt.Sprintf("run projector after change %v", recName)).
			AddEvent(cmdName).
			SetEventComment(cmdName, fmt.Sprintf("run projector after execute %v", cmdName)).
			AddEvent(objName).
			SetEventComment(objName, fmt.Sprintf("run projector after execute any command with parameter %v", objName)).
			SetWantErrors().
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
		})

		t.Run("must be ok obtain events map", func(t *testing.T) {
			events := prj.EventsMap()
			require.Len(events, 3)
			require.Contains(events, cmdName)
			require.EqualValues([]ProjectorEventKind{ProjectorEventKind_Execute}, events[cmdName])
			require.Contains(events, objName)
			require.EqualValues([]ProjectorEventKind{ProjectorEventKind_ExecuteWithParam}, events[objName])
			require.Contains(events, recName)
			require.EqualValues(ProjectorEventKind_AnyChanges, events[recName])
		})

		require.True(prj.WantErrors())

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

		require.False(p.WantErrors())
	})

	t.Run("projector validation errors", func(t *testing.T) {
		t.Run(" should error if empty events", func(t *testing.T) {
			apb := New()
			prj := apb.AddProjector(prjName)
			_, err := apb.Build()
			require.ErrorIs(err, ErrEmptyProjectorEvents)
			require.Contains(err.Error(), fmt.Sprint(prj))
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

	t.Run("panic if event type is not record, command or command parameter", func(t *testing.T) {
		apb := New()
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() { prj.AddEvent(QNameANY) })
	})

	t.Run("panic if event is incompatible with type", func(t *testing.T) {
		apb := New()
		_ = apb.AddCRecord(recName)
		_ = apb.AddObject(objName)
		_ = apb.AddCommand(cmdName).SetParam(objName)
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() { prj.AddEvent(recName, ProjectorEventKind_Execute) })
		require.Panics(func() { prj.AddEvent(objName, ProjectorEventKind_Update) })
		require.Panics(func() { prj.AddEvent(cmdName, ProjectorEventKind_ExecuteWithParam) })
	})

	t.Run("panic if comment unknown event", func(t *testing.T) {
		apb := New()
		prj := apb.AddProjector(NewQName("test", "projector"))
		require.Panics(func() {
			prj.SetEventComment(NewQName("test", "unknown"), "comment for unknown event should be panic")
		})
	})
}

func TestProjectorEventKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    ProjectorEventKind
		want string
	}{
		{name: `1 —> "ProjectorEventKind_Insert"`,
			k:    ProjectorEventKind_Insert,
			want: `ProjectorEventKind_Insert`,
		},
		{name: `2 —> "ProjectorEventKind_Update"`,
			k:    ProjectorEventKind_Update,
			want: `ProjectorEventKind_Update`,
		},
		{name: `ProjectorEventKind_Count —> <number>`,
			k:    ProjectorEventKind_Count,
			want: strconv.FormatUint(uint64(ProjectorEventKind_Count), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("ProjectorEventKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("ProjectorEventKind.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover ProjectorEventKind.String()", func(t *testing.T) {
		const tested = ProjectorEventKind_Count + 1
		want := "ProjectorEventKind(" + strconv.FormatInt(int64(tested), 10) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(ProjectorEventKind_Count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestProjectorEventKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    ProjectorEventKind
		want string
	}{
		{name: "basic", k: ProjectorEventKind_Update, want: "Update"},
		{name: "out of range", k: ProjectorEventKind_Count + 1, want: (ProjectorEventKind_Count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(ProjectorEventKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}

func TestProjectorEventKind_typeCompatible(t *testing.T) {
	type args struct {
		kind TypeKind
	}
	tests := []struct {
		name string
		i    ProjectorEventKind
		args args
		want bool
	}{
		// insert, update, deactivate
		{"ok Insert CDoc", ProjectorEventKind_Insert, args{TypeKind_CDoc}, true},
		{"ok Update WDoc", ProjectorEventKind_Update, args{TypeKind_WDoc}, true},
		{"ok Deactivate GDoc", ProjectorEventKind_Deactivate, args{TypeKind_GDoc}, true},

		{"fail Insert ODoc", ProjectorEventKind_Insert, args{TypeKind_ODoc}, false},
		{"fail Update ORecord", ProjectorEventKind_Update, args{TypeKind_ORecord}, false},
		{"fail Deactivate Object", ProjectorEventKind_Deactivate, args{TypeKind_Object}, false},

		// execute
		{"ok Execute Command", ProjectorEventKind_Execute, args{TypeKind_Command}, true},
		{"fail Execute CRecord", ProjectorEventKind_Execute, args{TypeKind_CRecord}, false},
		{"fail Execute Object", ProjectorEventKind_Execute, args{TypeKind_Object}, false},

		// execute with param
		{"ok Execute with Object", ProjectorEventKind_ExecuteWithParam, args{TypeKind_Object}, true},
		{"ok Execute with ODoc", ProjectorEventKind_ExecuteWithParam, args{TypeKind_ODoc}, true},
		{"fail Execute with ORecord", ProjectorEventKind_ExecuteWithParam, args{TypeKind_ORecord}, false},
		{"fail Execute with WRecord", ProjectorEventKind_ExecuteWithParam, args{TypeKind_WRecord}, false},

		// absurds
		{"fail Insert Query", ProjectorEventKind_Insert, args{TypeKind_Query}, false},
		{"fail Execute View", ProjectorEventKind_Execute, args{TypeKind_ViewRecord}, false},
		{"fail Execute with Workspace", ProjectorEventKind_ExecuteWithParam, args{TypeKind_Workspace}, false},

		{"fail out of bounds event", ProjectorEventKind_Count + 1, args{TypeKind_CDoc}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.i.typeCompatible(tt.args.kind); got != tt.want {
				t.Errorf("%v.typeCompatible(%v) = %v, want %v", tt.i, tt.args.kind, got, tt.want)
			}
		})
	}
}
