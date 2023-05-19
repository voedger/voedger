/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package imetrics

import (
	"github.com/voedger/voedger/pkg/istructs"
)

type IMetric interface {
	Name() string

	Vvm() string

	// App returns istructs.NullAppQName when not specified
	App() istructs.AppQName
}

type IMetrics interface {
	// Increase metric value with "delta".
	// The default metric value is always 0.
	// Naming best practices: https://prometheus.io/docs/practices/naming/
	//
	// @ConcurrentAccess
	Increase(metricName string, vvmName string, valueDelta float64)

	// Increase app metric value with "delta".
	// The default metric value is always 0.
	// Naming best practices: https://prometheus.io/docs/practices/naming/
	//
	// @ConcurrentAccess
	IncreaseApp(metricName string, vvmName string, app istructs.AppQName, valueDelta float64)

	// GetAll lists current values of all metrics
	//
	// @ConcurrentAccess
	List(cb func(metric IMetric, metricValue float64) (err error)) (err error)
}
