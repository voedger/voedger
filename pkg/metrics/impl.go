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

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type metric struct {
	name string
	app  appdef.AppQName
	vvm  string
}

func (m *metric) Name() string {
	return m.name
}

func (m *metric) Vvm() string {
	return m.vvm
}

func (m *metric) App() appdef.AppQName {
	return m.app
}

type mapMetrics struct {
	metrics map[metric]*MetricValue
	lock    sync.Mutex
}

func newMetrics() IMetrics {
	return &mapMetrics{
		metrics: make(map[metric]*MetricValue),
	}
}

func (m *mapMetrics) AppMetricAddr(metricName string, vvm string, app appdef.AppQName) *MetricValue {
	return m.get(metric{
		name: metricName,
		app:  app,
		vvm:  vvm,
	})
}

func (m *mapMetrics) MetricAddr(metricName string, vvmName string) *MetricValue {
	return m.get(metric{
		name: metricName,
		app:  istructs.AppQName_null,
		vvm:  vvmName,
	})
}

func (m *mapMetrics) Increase(metricName string, vvm string, valueDelta float64) {
	m.MetricAddr(metricName, vvm).Increase(valueDelta)
}

func (m *mapMetrics) IncreaseApp(metricName string, vvm string, app appdef.AppQName, valueDelta float64) {
	m.AppMetricAddr(metricName, vvm, app).Increase(valueDelta)
}

func (m *mapMetrics) get(key metric) *MetricValue {
	m.lock.Lock()
	defer m.lock.Unlock()
	if mv, ok := m.metrics[key]; ok {
		return mv
	}
	value := MetricValue(0)
	m.metrics[key] = &value
	return &value
}

func (m *mapMetrics) List(cb func(metric IMetric, metricValue float64) (err error)) (err error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for metric, value := range m.metrics {
		ptr := (*uint64)(unsafe.Pointer(value))
		err = cb(&metric, math.Float64frombits(atomic.LoadUint64(ptr)))
		if err != nil {
			return
		}
	}
	return err
}

func ToPrometheus(metric IMetric, metricValue float64) []byte {
	bb := bytes.Buffer{}
	bb.WriteString(metric.Name())
	if metric.App() != appdef.NullAppQName || metric.Vvm() != "" {
		bb.WriteRune('{')
		if metric.App() != appdef.NullAppQName {
			bb.WriteString(`app="`)
			bb.WriteString(metric.App().String())
			bb.WriteRune('"')
		}
		if metric.Vvm() != "" {
			if metric.App() != appdef.NullAppQName {
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
