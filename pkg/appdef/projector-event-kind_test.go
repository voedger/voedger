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
