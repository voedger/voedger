/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func TestExtensionEngineKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.ExtensionEngineKind
		want string
	}{
		{name: `0 —> "ExtensionEngineKind_null"`,
			k:    appdef.ExtensionEngineKind_null,
			want: `ExtensionEngineKind_null`,
		},
		{name: `1 —> "ExtensionEngineKind_BuiltIn"`,
			k:    appdef.ExtensionEngineKind_BuiltIn,
			want: `ExtensionEngineKind_BuiltIn`,
		},
		{name: `ExtensionEngineKind_count —> <number>`,
			k:    appdef.ExtensionEngineKind_count,
			want: utils.UintToString(appdef.ExtensionEngineKind_count),
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
		const tested = appdef.ExtensionEngineKind_count + 1
		want := "ExtensionEngineKind(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(ExtensionEngineKind_count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestExtensionEngineKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.ExtensionEngineKind
		want string
	}{
		{name: "basic", k: appdef.ExtensionEngineKind_BuiltIn, want: "BuiltIn"},
		{name: "out of range", k: appdef.ExtensionEngineKind_count + 1, want: (appdef.ExtensionEngineKind_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(ExtensionEngineKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
