/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 */

package imetrics

import (
	"bytes"
	"strconv"
	"sync"

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
	metrics map[metric]float64
	lock    sync.Mutex
}

func newMetrics() IMetrics {
	return &mapMetrics{
		metrics: make(map[metric]float64),
	}
}

func (m *mapMetrics) Increase(metricName string, vvm string, valueDelta float64) {
	key := metric{
		name: metricName,
		app:  istructs.AppQName_null,
		vvm:  vvm,
	}
	m.increase(key, valueDelta)
}

func (m *mapMetrics) IncreaseApp(metricName string, vvm string, app istructs.AppQName, valueDelta float64) {
	key := metric{
		name: metricName,
		app:  app,
		vvm:  vvm,
	}
	m.increase(key, valueDelta)
}

func (m *mapMetrics) increase(key metric, valueDelta float64) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.metrics[key] = m.metrics[key] + valueDelta
}

func (m *mapMetrics) List(cb func(metric IMetric, metricValue float64) (err error)) (err error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for metric, value := range m.metrics {
		err = cb(&metric, value)
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
