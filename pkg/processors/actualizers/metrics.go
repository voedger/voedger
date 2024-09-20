/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package actualizers

import (
	"fmt"
	"strings"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

const (
	ProjectorsInError = "voedger_projectors_in_error"

	// internal metrics
	aaFlushesTotal  = "voedger_aa_flushes_total"
	aaCurrentOffset = "voedger_aa_current_offset"
	aaStoredOffset  = "voedger_aa_stored_offset"
)

type simpleMetrics struct {
	mx sync.RWMutex
	v  map[string]int64
}

func newSimpleMetrics() *simpleMetrics {
	return &simpleMetrics{v: make(map[string]int64)}
}

func (*simpleMetrics) key(metricName string, partition istructs.PartitionID, projection appdef.QName) string {
	return fmt.Sprintf("%s:%d:%s", metricName, partition, projection)
}

func (m *simpleMetrics) total(metricName string) int64 {
	key := metricName + ":"
	value := int64(0)

	m.mx.RLock()
	defer m.mx.RUnlock()
	for k, v := range m.v {
		if strings.HasPrefix(k, key) {
			value += v
		}
	}
	return value
}

func (m *simpleMetrics) value(metricName string, partition istructs.PartitionID, projection appdef.QName) int64 {
	k := m.key(metricName, partition, projection)

	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.v[k]
}

func (m *simpleMetrics) Increase(metricName string, partition istructs.PartitionID, projection appdef.QName, valueDelta float64) {
	k := m.key(metricName, partition, projection)

	m.mx.Lock()
	defer m.mx.Unlock()
	if v, ok := m.v[k]; ok {
		m.v[k] = v + int64(valueDelta)
	} else {
		m.v[k] = int64(valueDelta)
	}
}

func (m *simpleMetrics) Set(metricName string, partition istructs.PartitionID, projection appdef.QName, value float64) {
	k := m.key(metricName, partition, projection)

	m.mx.Lock()
	defer m.mx.Unlock()
	m.v[k] = int64(value)
}
