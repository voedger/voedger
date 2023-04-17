/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 */

package ihttpimpl

import (
	"net/http"
	"strconv"

	"github.com/voedger/voedger/pkg/ibus"
	"github.com/voedger/voedger/pkg/ihttp"
)

func NewProcessor(params ihttp.CLIParams, bus ibus.IBus) (server ihttp.IHTTPProcessor, cleanup func(), err error) {
	port := strconv.Itoa(params.Port)
	r := &router{}
	httpProcessor := httpProcessor{
		params: params,
		router: r,
		server: &http.Server{
			Addr:              ":" + port,
			Handler:           r,
			ReadHeaderTimeout: defaultReadHeaderTimeout,
		},
		bus: bus,
	}
	httpProcessor.bus.RegisterReceiver("sys", "HTTPProcessor", 0, "c", httpProcessor.Receiver, NumOfAPIProcessors, APIChannelBufferSize)
	return &httpProcessor, httpProcessor.cleanup, err
}

func NewAPI(bus ibus.IBus, httpProcessor ihttp.IHTTPProcessor) (controller ihttp.IHTTPProcessorAPI, err error) {
	sender, ok := bus.QuerySender("sys", "HTTPProcessor", 0, "c")
	if !ok {
		panic("httpProcessorControllerFactory: sender not found")
	}
	return &processorAPI{senderHttp: sender}, err
}
