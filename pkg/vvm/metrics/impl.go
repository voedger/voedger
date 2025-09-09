/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package metrics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"

	imetrics "github.com/voedger/voedger/pkg/metrics"
)

func (ms *metricsService) Prepare(interface{}) (err error) {
	ms.listener, err = net.Listen("tcp", coreutils.ListenAddr(ms.port))
	return err
}

func (ms *metricsService) Run(_ context.Context) {
	logger.Info("Starting Metrics Service on", ms.listener.Addr().(*net.TCPAddr).String())
	if err := ms.Serve(ms.listener); !errors.Is(err, http.ErrServerClosed) {
		panic("metrics service failure: " + err.Error())
	}
}

func (ms *metricsService) Stop() {
	// context here is used to avoid infinite awaiting for all connections close
	// we want to all connections to close, so we provide context which is not cancelled
	if err := ms.Shutdown(context.Background()); err != nil {
		logger.Error("metrics service shutdown failed: ", err)
	}
}

func (ms *metricsService) GetPort() int {
	return ms.listener.Addr().(*net.TCPAddr).Port
}

func provideHandler(metrics imetrics.IMetrics) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		isCheck := r.URL.Path == "/metrics/check"

		if (r.URL.Path != "/metrics" && !isCheck) || r.Method != http.MethodGet {
			http.Error(rw, "404 not found", http.StatusNotFound)
			return
		}

		if isCheck {
			rw.WriteHeader(http.StatusOK)
			if _, err := rw.Write([]byte("ok")); err != nil {
				logger.Error("metrics service: failed to reply check ok: ", err)
			}
			return
		}

		err := metrics.List(func(metric imetrics.IMetric, metricValue float64) (err error) {
			if _, err = rw.Write(imetrics.ToPrometheus(metric, metricValue)); err != nil {
				return fmt.Errorf("metrics service: failed to write metric %s for app %s on VVM %s: %w", metric.Name(), metric.App(), metric.Vvm(), err)
			}
			return
		})
		if err != nil {
			logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
		}
	}
}
