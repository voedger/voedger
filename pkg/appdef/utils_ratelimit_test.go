/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func TestRateScopeTrimString(t *testing.T) {
	tests := []struct {
		name string
		s    RateScope
		want string
	}{
		{name: "basic", s: RateScope_AppPartition, want: "AppPartition"},
		{name: "out of range", s: RateScope_count + 1, want: (RateScope_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.TrimString(); got != tt.want {
				t.Errorf("%v.(RateScope).TrimString() = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestLimitOption_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		o    LimitOption
		want string
	}{
		{name: `0 —> "LimitOption_ALL"`,
			o:    LimitOption_ALL,
			want: `LimitOption_ALL`,
		},
		{name: `1 —> "LimitOption_EACH"`,
			o:    LimitOption_EACH,
			want: `LimitOption_EACH`,
		},
		{name: `LimitOption_count —> <number>`,
			o:    LimitOption_count,
			want: utils.UintToString(LimitOption_count),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.o.MarshalText()
			if err != nil {
				t.Errorf("LimitOption.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("LimitOption.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover LimitOption.String()", func(t *testing.T) {
		const tested = LimitOption_count + 1
		want := "LimitOption(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(LimitOption_count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestLimitOptionTrimString(t *testing.T) {
	tests := []struct {
		name string
		o    LimitOption
		want string
	}{
		{name: "basic", o: LimitOption_ALL, want: "ALL"},
		{name: "out of range", o: LimitOption_count + 1, want: (LimitOption_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.TrimString(); got != tt.want {
				t.Errorf("%v.(LimitOption).TrimString() = %v, want %v", tt.o, got, tt.want)
			}
		})
	}
}
