/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package metrics

func MetricCollectors() IMetricCollectors {
	return &metricCollectors{}
}

func MetricReporters() IMetricReporters {
	return &metricReporters{}
}
