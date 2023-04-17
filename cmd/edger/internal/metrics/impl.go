/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metrics

import "context"

type metricCollectors struct{}

func (mc *metricCollectors) CollectMetrics(context.Context) (*Metrics, error) {
	//TODO: real collecting
	m := Metrics{}
	return &m, nil
}

type metricReporters struct{}

func (mr *metricReporters) ReportMetrics(context.Context, *Metrics) error {
	//TODO: real reporting
	return nil
}
