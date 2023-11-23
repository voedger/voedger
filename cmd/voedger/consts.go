/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 */

package main

import "github.com/voedger/voedger/pkg/ihttp"

const (
	Default_ihttp_Port                      = 80
	Default_ibus_MaxNumOfConcurrentRequests = 1000
	Default_ibus_ReadWriteTimeoutNS         = 5_000_000_000
)

const (
	defaultGrafanaPort    ihttp.GrafanaPort    = 3000
	defaultPrometheusPort ihttp.PrometheusPort = 9090
)
