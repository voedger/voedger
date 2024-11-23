/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func TestFilterKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    FilterKind
		want string
	}{
		{name: `0 —> "FilterKind_null"`,
			k:    FilterKind_null,
			want: `FilterKind_null`,
		},
		{name: `1 —> "FilterKind_QNames"`,
			k:    FilterKind_QNames,
			want: `FilterKind_QNames`,
		},
		{name: `FilterKind_count —> <number>`,
			k:    FilterKind_count,
			want: utils.UintToString(FilterKind_count),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("FilterKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("FilterKind.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover FilterKind.String()", func(t *testing.T) {
		const tested = FilterKind_count + 1
		want := "FilterKind(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(FilterKind_count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestFilterKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    FilterKind
		want string
	}{
		{name: "basic", k: FilterKind_QNames, want: "QNames"},
		{name: "out of range", k: FilterKind_count + 1, want: (FilterKind_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(FilterKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
