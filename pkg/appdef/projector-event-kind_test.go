/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author Maxim Geraskin
 */

package appdef

import (
	"strconv"
	"testing"
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
		{"ok Insert CDoc", ProjectorEventKind_Insert, args{TypeKind_CDoc}, true},
		{"ok Update CDoc", ProjectorEventKind_Update, args{TypeKind_CDoc}, true},
		{"ok Deactivate CDoc", ProjectorEventKind_Deactivate, args{TypeKind_CDoc}, true},

		{"ok Insert WDoc", ProjectorEventKind_Insert, args{TypeKind_WDoc}, true},
		{"ok Update WDoc", ProjectorEventKind_Update, args{TypeKind_WDoc}, true},
		{"ok Deactivate WDoc", ProjectorEventKind_Deactivate, args{TypeKind_WDoc}, true},

		{"ok Insert ODoc", ProjectorEventKind_Insert, args{TypeKind_ODoc}, true},
		{"fail Update ODoc", ProjectorEventKind_Update, args{TypeKind_ODoc}, false},
		{"fail Deactivate ODoc", ProjectorEventKind_Deactivate, args{TypeKind_ODoc}, false},

		{"ok Execute Command", ProjectorEventKind_Execute, args{TypeKind_Command}, true},
		{"fail Execute GRecord", ProjectorEventKind_Execute, args{TypeKind_GRecord}, false},

		{"fail Insert Command", ProjectorEventKind_Insert, args{TypeKind_Command}, false},

		{"fail Insert Query", ProjectorEventKind_Insert, args{TypeKind_Query}, false},
		{"fail Execute View", ProjectorEventKind_Execute, args{TypeKind_ViewRecord}, false},

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
