/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func TestRateScopeTrimString(t *testing.T) {
	tests := []struct {
		name string
		s    appdef.RateScope
		want string
	}{
		{name: "basic", s: appdef.RateScope_AppPartition, want: "AppPartition"},
		{name: "out of range", s: appdef.RateScope_count + 1, want: (appdef.RateScope_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.TrimString(); got != tt.want {
				t.Errorf("%v.(RateScope).TrimString() = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func Test_LimitFilterOption_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		o    appdef.LimitFilterOption
		want string
	}{
		{name: `0 —> "LimitFilterOption_ALL"`,
			o:    appdef.LimitFilterOption_ALL,
			want: `LimitFilterOption_ALL`,
		},
		{name: `1 —> "LimitFilterOption_EACH"`,
			o:    appdef.LimitFilterOption_EACH,
			want: `LimitFilterOption_EACH`,
		},
		{name: `LimitFilterOption_count —> <number>`,
			o:    appdef.LimitFilterOption_count,
			want: utils.UintToString(appdef.LimitFilterOption_count),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.o.MarshalText()
			if err != nil {
				t.Errorf("LimitFilterOption.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("LimitFilterOption.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover LimitFilterOption.String()", func(t *testing.T) {
		const tested = appdef.LimitFilterOption_count + 1
		want := "LimitFilterOption(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(LimitFilterOption_count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestLimitFilterOptionTrimString(t *testing.T) {
	tests := []struct {
		name string
		o    appdef.LimitFilterOption
		want string
	}{
		{name: "basic", o: appdef.LimitFilterOption_ALL, want: "ALL"},
		{name: "out of range", o: appdef.LimitFilterOption_count + 1, want: (appdef.LimitFilterOption_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.TrimString(); got != tt.want {
				t.Errorf("%v.(LimitFilterOption).TrimString() = %v, want %v", tt.o, got, tt.want)
			}
		})
	}
}
