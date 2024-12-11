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

func TestLimitOption_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		o    appdef.LimitOption
		want string
	}{
		{name: `0 —> "LimitOption_ALL"`,
			o:    appdef.LimitOption_ALL,
			want: `LimitOption_ALL`,
		},
		{name: `1 —> "LimitOption_EACH"`,
			o:    appdef.LimitOption_EACH,
			want: `LimitOption_EACH`,
		},
		{name: `LimitOption_count —> <number>`,
			o:    appdef.LimitOption_count,
			want: utils.UintToString(appdef.LimitOption_count),
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
		const tested = appdef.LimitOption_count + 1
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
		o    appdef.LimitOption
		want string
	}{
		{name: "basic", o: appdef.LimitOption_ALL, want: "ALL"},
		{name: "out of range", o: appdef.LimitOption_count + 1, want: (appdef.LimitOption_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.TrimString(); got != tt.want {
				t.Errorf("%v.(LimitOption).TrimString() = %v, want %v", tt.o, got, tt.want)
			}
		})
	}
}
