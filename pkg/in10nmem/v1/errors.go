/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 */

package in10nmemv1

import "errors"

var (
	ErrChannelExpired      = errors.New("channel time to live expired")
	ErrMetricDoesNotExists = errors.New("metric does not exists")
)
