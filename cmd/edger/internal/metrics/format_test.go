/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metrics

import (
	"fmt"
	"testing"
)

func TestMetricKind_String(t *testing.T) {
	tests := []struct {
		name string
		kind MetricKind
		want string
	}{
		{"Counter", Counter, longMetricKindNames[0]},
		{"Gauge", Gauge, longMetricKindNames[1]},
		{"out of bounds", LastMetricKind, fmt.Sprintf("?MetricKind(%d)", LastMetricKind)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("MetricKind(%d).String() = %v, want %v", tt.kind, got, tt.want)
			}
		})
	}
}
