/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 */

package ihttpimpl

import (
	"net/http"
	"strconv"
	"time"

	"github.com/voedger/voedger/pkg/ihttp"
)

func NewProcessor(params ihttp.CLIParams) (server ihttp.IHTTPProcessor, cleanup func(), err error) {
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
		readWriteTimeout:    ReadWriteTimeout,
		addressHandlersMap:  make(map[addressType]*addressHandlerType),
		requestContextsPool: make(chan *requestContextType, MaxNumOfConcurrentRequests),
	}
	for i := 0; i < MaxNumOfConcurrentRequests; i++ {
		requestContext := requestContextType{
			errReached:      true,
			responseChannel: make(responseChannelType, ResponseChannelBufferSize),
			senderTimer:     time.NewTimer(ReadWriteTimeout),
		}
		httpProcessor.requestContextsPool <- &requestContext
	}
	httpProcessor.RegisterReceiver("sys", "HTTPProcessor", 0, "c", httpProcessor.Receiver, NumOfAPIProcessors, APIChannelBufferSize)
	return &httpProcessor, httpProcessor.cleanup, err
}

func NewAPI(httpProcessor ihttp.IHTTPProcessor) (controller ihttp.IHTTPProcessorAPI, err error) {
	sender, ok := httpProcessor.QuerySender("sys", "HTTPProcessor", 0, "c")
	if !ok {
		panic("httpProcessorControllerFactory: sender not found")
	}
	return &processorAPI{senderHttp: sender}, err
}
