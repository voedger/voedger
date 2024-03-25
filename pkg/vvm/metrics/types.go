/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package metrics

import (
	"net"
	"net/http"

	"github.com/voedger/voedger/pkg/pipeline"
)

type MetricsServicePort int
type MetricsService pipeline.IService

type metricsService struct {
	pipeline.NOPService
	*http.Server
	listener net.Listener
	port     int
}
