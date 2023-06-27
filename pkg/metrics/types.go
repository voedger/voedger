/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package imetrics

import (
	"math"
	"sync/atomic"
	"unsafe"
)

type MetricsFactory func() IMetrics

type MetricValue float64

func (m *MetricValue) Increase(delta float64) {
	var swapped bool
	ptr := (*uint64)(unsafe.Pointer(m))
	for !swapped {
		old := math.Float64frombits(atomic.LoadUint64(ptr))
		new := old + delta
		swapped = atomic.CompareAndSwapUint64(
			ptr,
			math.Float64bits(old),
			math.Float64bits(new),
		)
	}
}
