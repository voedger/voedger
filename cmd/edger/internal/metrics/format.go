/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
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
