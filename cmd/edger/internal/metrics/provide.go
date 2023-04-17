/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metrics

func MetricCollectors() IMetricCollectors {
	return &metricCollectors{}
}

func MetricReporters() IMetricReporters {
	return &metricReporters{}
}
