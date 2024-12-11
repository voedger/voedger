/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"
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
