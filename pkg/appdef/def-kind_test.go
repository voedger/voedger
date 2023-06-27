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

func TestDefKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    DefKind
		want string
	}{
		{name: "basic", k: DefKind_CDoc, want: "CDoc"},
		{name: "out of range", k: DefKind_FakeLast + 1, want: (DefKind_FakeLast + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(DefKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
