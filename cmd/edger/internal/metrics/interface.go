/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
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
