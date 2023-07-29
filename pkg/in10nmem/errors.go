/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree. 
 */

package in10nmem

import "errors"

var (
	ErrChannelExpired      = errors.New("channel time to live expired")
	ErrMetricDoesNotExists = errors.New("metric does not exists")
)
