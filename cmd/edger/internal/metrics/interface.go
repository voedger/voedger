/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metrics

import (
	"context"
	"time"
)

type (
	MetricKind uint8

	MetricSample struct {
		Value float64
		Time  time.Time
	}

	Metric struct {
		Name    string
		Labels  map[string]string
		Kind    MetricKind
		Samples []MetricSample
	}

	Metrics []Metric
)

type IMetricCollectors interface {
	CollectMetrics(context.Context) (*Metrics, error)
}

type IMetricReporters interface {
	ReportMetrics(context.Context, *Metrics) error
}
