/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

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
		{name: `ProjectorEventKind_count —> <number>`,
			k:    ProjectorEventKind_count,
			want: utils.UintToString(ProjectorEventKind_count),
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
		const tested = ProjectorEventKind_count + 1
		want := "ProjectorEventKind(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(ProjectorEventKind_count + 1).String() = %v, want %v", got, want)
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
		{name: "out of range", k: ProjectorEventKind_count + 1, want: (ProjectorEventKind_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(ProjectorEventKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}

func Test_projectorEventCompatableWith(t *testing.T) {
	testName := NewQName("test", "test")

	type typ struct {
		kind TypeKind
		name QName
	}
	tests := []struct {
		name  string
		event ProjectorEventKind
		typ   typ
		want  bool
	}{
		// insert, update, deactivate
		{"ok Insert CDoc", ProjectorEventKind_Insert, typ{TypeKind_CDoc, testName}, true},
		{"ok Update WDoc", ProjectorEventKind_Update, typ{TypeKind_WDoc, testName}, true},
		{"ok Deactivate GDoc", ProjectorEventKind_Deactivate, typ{TypeKind_GDoc, testName}, true},

		{"fail Insert ODoc", ProjectorEventKind_Insert, typ{TypeKind_ODoc, testName}, false},
		{"fail Update ORecord", ProjectorEventKind_Update, typ{TypeKind_ORecord, testName}, false},
		{"fail Deactivate Object", ProjectorEventKind_Deactivate, typ{TypeKind_Object, testName}, false},

		// execute
		{"ok Execute Command", ProjectorEventKind_Execute, typ{TypeKind_Command, testName}, true},
		{"fail Execute CRecord", ProjectorEventKind_Execute, typ{TypeKind_CRecord, testName}, false},
		{"fail Execute Object", ProjectorEventKind_Execute, typ{TypeKind_Object, testName}, false},

		// execute with param
		{"ok Execute with Object", ProjectorEventKind_ExecuteWithParam, typ{TypeKind_Object, testName}, true},
		{"ok Execute with ODoc", ProjectorEventKind_ExecuteWithParam, typ{TypeKind_ODoc, testName}, true},
		{"fail Execute with ORecord", ProjectorEventKind_ExecuteWithParam, typ{TypeKind_ORecord, testName}, false},
		{"fail Execute with WRecord", ProjectorEventKind_ExecuteWithParam, typ{TypeKind_WRecord, testName}, false},

		// ANY
		{"ok Insert any record", ProjectorEventKind_Insert, typ{TypeKind_Any, QNameAnyRecord}, true},
		{"ok Update any WDoc", ProjectorEventKind_Update, typ{TypeKind_Any, QNameAnyWDoc}, true},
		{"ok Deactivate any GDoc", ProjectorEventKind_Deactivate, typ{TypeKind_Any, QNameAnyGDoc}, true},
		{"ok Execute any Command", ProjectorEventKind_Execute, typ{TypeKind_Any, QNameAnyCommand}, true},
		{"ok Execute with any ODoc", ProjectorEventKind_ExecuteWithParam, typ{TypeKind_Any, QNameAnyODoc}, true},
		{"fail Insert any type", ProjectorEventKind_Insert, typ{TypeKind_Any, QNameANY}, false},
		{"fail Execute any Object", ProjectorEventKind_Execute, typ{TypeKind_Any, QNameAnyObject}, false},
		{"fail Execute with any View", ProjectorEventKind_ExecuteWithParam, typ{TypeKind_Any, QNameAnyView}, false},

		// absurds
		{"fail Insert Query", ProjectorEventKind_Insert, typ{TypeKind_Query, testName}, false},
		{"fail Execute View", ProjectorEventKind_Execute, typ{TypeKind_ViewRecord, testName}, false},
		{"fail Execute with Workspace", ProjectorEventKind_ExecuteWithParam, typ{TypeKind_Workspace, testName}, false},

		{"fail out of bounds event", ProjectorEventKind_count + 1, typ{TypeKind_CDoc, testName}, false},
	}

	require := require.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := new(mockType)
			typ.kind = tt.typ.kind
			typ.name = tt.typ.name
			if got, err := projectorEventCompatableWith(tt.event, typ); got != tt.want {
				t.Errorf("projectorEventCompatableWith(%v, {%v «%v»}) = %v, want %v", tt.event, tt.typ.kind.TrimString(), tt.typ.name, got, tt.want)
				if !tt.want {
					require.Error(err, require.Is(ErrIncompatibleError), require.Has(tt.event), require.Has(tt.typ.name))
				}
			}
		})
	}
}
