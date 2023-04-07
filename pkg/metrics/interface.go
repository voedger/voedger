/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package imetrics

import (
	"github.com/untillpro/voedger/pkg/istructs"
)

type IMetric interface {
	Name() string

	Hvm() string

	// App returns istructs.NullAppQName when not specified
	App() istructs.AppQName
}

type IMetrics interface {
	// Increase metric value with "delta".
	// The default metric value is always 0.
	// Naming best practices: https://prometheus.io/docs/practices/naming/
	//
	// @ConcurrentAccess
	Increase(metricName string, hvm string, valueDelta float64)

	// Increase app metric value with "delta".
	// The default metric value is always 0.
	// Naming best practices: https://prometheus.io/docs/practices/naming/
	//
	// @ConcurrentAccess
	IncreaseApp(metricName string, hvm string, app istructs.AppQName, valueDelta float64)

	// GetAll lists current values of all metrics
	//
	// @ConcurrentAccess
	List(cb func(metric IMetric, metricValue float64) (err error)) (err error)
}
