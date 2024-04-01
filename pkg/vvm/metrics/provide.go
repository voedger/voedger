/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package metrics

import (
	"context"
	"net"
	"net/http"
	"time"

	imetrics "github.com/voedger/voedger/pkg/metrics"
	router2 "github.com/voedger/voedger/pkg/router"
)

func ProvideMetricsService(vvmCtx context.Context, metricsServicePort MetricsServicePort, imetrics imetrics.IMetrics) MetricsService {
	return &metricsService{
		Server: &http.Server{
			Handler: provideHandler(imetrics),
			BaseContext: func(l net.Listener) context.Context {
				return vvmCtx
			},
			ReadHeaderTimeout: router2.DefaultRouterReadTimeout * time.Second, // avoiding potential Slowloris attack (G112 linter rule)
		},
		port: int(metricsServicePort),
	}
}
