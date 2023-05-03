/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package metrics

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	router2 "github.com/untillpro/airs-router2"
	imetrics "github.com/voedger/voedger/pkg/metrics"
)

func ProvideMetricsService(hvmCtx context.Context, metricsServicePort MetricsServicePort, imetrics imetrics.IMetrics) MetricsService {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(int(metricsServicePort)))
	if err != nil {
		panic(err)
	}

	return &metricsService{
		Server: &http.Server{
			Handler: provideHandler(imetrics),
			BaseContext: func(l net.Listener) context.Context {
				return hvmCtx
			},
			ReadHeaderTimeout: router2.DefaultRouterReadTimeout * time.Second, // avoiding potential Slowloris attack (G112 linter rule)
		},
		listener: listener,
	}
}
