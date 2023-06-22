/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 */

package imetrics

import (
	"bytes"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/voedger/voedger/pkg/istructs"
)

type metric struct {
	name string
	app  istructs.AppQName
	vvm  string
}

func (m *metric) Name() string {
	return m.name
}

func (m *metric) Vvm() string {
	return m.vvm
}

func (m *metric) App() istructs.AppQName {
	return m.app
}

type mapMetrics struct {
	metrics map[metric]*float64
	lock    sync.Mutex
}

func newMetrics() IMetrics {
	return &mapMetrics{
		metrics: make(map[metric]*float64),
	}
}

func (m *mapMetrics) AppMetricAddr(metricName string, vvm string, app istructs.AppQName) *float64 {
	return m.get(metric{
		name: metricName,
		app:  app,
		vvm:  vvm,
	})
}

func (m *mapMetrics) MetricAddr(metricName string, vvmName string) *float64 {
	return m.get(metric{
		name: metricName,
		app:  istructs.AppQName_null,
		vvm:  vvmName,
	})
}

func (m *mapMetrics) Increase(metricName string, vvm string, valueDelta float64) {
	mv := m.MetricAddr(metricName, vvm)
	AddFloat64(mv, valueDelta)
}

func (m *mapMetrics) IncreaseApp(metricName string, vvm string, app istructs.AppQName, valueDelta float64) {
	mv := m.AppMetricAddr(metricName, vvm, app)
	AddFloat64(mv, valueDelta)
}

func (m *mapMetrics) get(key metric) *float64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	if mv, ok := m.metrics[key]; ok {
		return mv
	}
	value := float64(0)
	m.metrics[key] = &value
	return &value

}

func AddFloat64(val *float64, delta float64) {
	var swapped bool
	for !swapped {
		old := *val
		new := old + delta
		swapped = atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(val)),
			math.Float64bits(old),
			math.Float64bits(new),
		)
	}
}

func (m *mapMetrics) List(cb func(metric IMetric, metricValue float64) (err error)) (err error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for metric, value := range m.metrics {
		err = cb(&metric, *value)
		if err != nil {
			return
		}
	}
	return err
}

func ToPrometheus(metric IMetric, metricValue float64) []byte {
	bb := bytes.Buffer{}
	bb.WriteString(metric.Name())
	if metric.App() != istructs.NullAppQName || metric.Vvm() != "" {
		bb.WriteRune('{')
		if metric.App() != istructs.NullAppQName {
			bb.WriteString(`app="`)
			bb.WriteString(metric.App().String())
			bb.WriteRune('"')
		}
		if metric.Vvm() != "" {
			if metric.App() != istructs.NullAppQName {
				bb.WriteRune(',')
			}
			bb.WriteString(`vvm="`)
			bb.WriteString(metric.Vvm())
			bb.WriteRune('"')
		}
		bb.WriteRune('}')
	}
	bb.WriteRune(' ')
	bb.WriteString(strconv.FormatFloat(metricValue, 'f', -1, bitSize))
	bb.WriteRune('\n')
	return bb.Bytes()
}
