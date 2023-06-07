/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strconv"
	"testing"
)

func TestExtensionEngineKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    ExtensionEngineKind
		want string
	}{
		{name: `0 —> "ExtensionEngineKind_null"`,
			k:    ExtensionEngineKind_null,
			want: `ExtensionEngineKind_null`,
		},
		{name: `1 —> "ExtensionEngineKind_BuiltIn"`,
			k:    ExtensionEngineKind_BuiltIn,
			want: `ExtensionEngineKind_BuiltIn`,
		},
		{name: `ExtensionEngineKind_FakeLast —> <number>`,
			k:    ExtensionEngineKind_FakeLast,
			want: strconv.FormatUint(uint64(ExtensionEngineKind_FakeLast), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("ExtensionEngineKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("ExtensionEngineKind.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover ExtensionEngineKind.String()", func(t *testing.T) {
		const tested = ExtensionEngineKind_FakeLast + 1
		want := "ExtensionEngineKind(" + strconv.FormatInt(int64(tested), 10) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(ExtensionEngineKind_FakeLast + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestExtensionEngineKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    ExtensionEngineKind
		want string
	}{
		{name: "basic", k: ExtensionEngineKind_BuiltIn, want: "BuiltIn"},
		{name: "out of range", k: ExtensionEngineKind_FakeLast + 1, want: (ExtensionEngineKind_FakeLast + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(ExtensionEngineKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
