/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strconv"
	"testing"
)

func TestDefKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    DefKind
		want string
	}{
		{name: `0 —> "DefKind_null"`,
			k:    DefKind_null,
			want: `DefKind_null`,
		},
		{name: `1 —> "DefKind_GDoc"`,
			k:    DefKind_GDoc,
			want: `DefKind_GDoc`,
		},
		{name: `DefKind_FakeLast —> <number>`,
			k:    DefKind_FakeLast,
			want: strconv.FormatUint(uint64(DefKind_FakeLast), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("DefKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("DefKind.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover DefKind.String()", func(t *testing.T) {
		const tested = DefKind_FakeLast + 1
		want := "DefKind(" + strconv.FormatInt(int64(tested), 10) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(DefKind_FakeLast + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestDefKindToString(t *testing.T) {
	tests := []struct {
		name string
		k    DefKind
		want string
	}{
		{name: "vulgaris", k: DefKind_CDoc, want: "CDoc"},
		{name: "out of range", k: DefKind_FakeLast + 1, want: (DefKind_FakeLast + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.ToString(); got != tt.want {
				t.Errorf("%v.(DefKind).ToString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}

func TestDefKind_IsStructure(t *testing.T) {
	tests := []struct {
		name string
		k    DefKind
		want bool
	}{
		{"document", DefKind_CDoc, true},
		{"record", DefKind_CRecord, true},
		{"object", DefKind_Object, true},
		{"element", DefKind_Element, true},
		{"view", DefKind_ViewRecord_Value, false},
		{"resource", DefKind_CommandFunction, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.IsStructure(); got != tt.want {
				t.Errorf("%s: %v.IsStructure() = %v, want %v", tt.name, tt.k, got, tt.want)
			}
		})
	}
}

func TestDefKind_UniquesAvailable(t *testing.T) {
	tests := []struct {
		name string
		k    DefKind
		want bool
	}{
		{"document", DefKind_CDoc, true},
		{"record", DefKind_CRecord, true},
		{"element", DefKind_Element, true},
		{"view", DefKind_ViewRecord_Value, false},
		{"resource", DefKind_CommandFunction, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.UniquesAvailable(); got != tt.want {
				t.Errorf("%s: %v.UniquesAvailable() = %v, want %v", tt.name, tt.k, got, tt.want)
			}
		})
	}
}
