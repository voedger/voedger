/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strconv"
	"testing"
)

func TestTypeKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    TypeKind
		want string
	}{
		{name: `0 —> "TypeKind_null"`,
			k:    TypeKind_null,
			want: `TypeKind_null`,
		},
		{name: `1 —> "TypeKind_GDoc"`,
			k:    TypeKind_GDoc,
			want: `TypeKind_GDoc`,
		},
		{name: `TypeKind_FakeLast —> <number>`,
			k:    TypeKind_FakeLast,
			want: strconv.FormatUint(uint64(TypeKind_FakeLast), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("TypeKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("TypeKind.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover TypeKind.String()", func(t *testing.T) {
		const tested = TypeKind_FakeLast + 1
		want := "TypeKind(" + strconv.FormatInt(int64(tested), 10) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(TypeKind_FakeLast + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestTypeKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    TypeKind
		want string
	}{
		{name: "basic", k: TypeKind_CDoc, want: "CDoc"},
		{name: "out of range", k: TypeKind_FakeLast + 1, want: (TypeKind_FakeLast + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(TypeKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
