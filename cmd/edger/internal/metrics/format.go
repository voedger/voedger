/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metrics

import "fmt"

var longMetricKindNames = [LastMetricKind]string{
	"Counter",
	"Gauge",
}

func (k MetricKind) String() string {
	const badResultFmt = "?MetricKind(%d)"
	if k < LastMetricKind {
		return longMetricKindNames[k]
	}
	return fmt.Sprintf(badResultFmt, k)
}
