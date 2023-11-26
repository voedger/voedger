/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 */

package ihttpimpl

import (
	"net/http"
	"strconv"

	"github.com/voedger/voedger/pkg/ihttp"
)

func NewProcessor(params ihttp.CLIParams) (server ihttp.IHTTPProcessor, cleanup func(), err error) {
	port := strconv.Itoa(params.Port)
	r := newRouter()
	httpProcessor := httpProcessor{
		params: params,
		router: r,
		server: &http.Server{
			Addr:              ":" + port,
			Handler:           r,
			ReadHeaderTimeout: defaultReadHeaderTimeout,
		},
	}
	return &httpProcessor, httpProcessor.cleanup, err
}

func NewAPI(httpProcessor ihttp.IHTTPProcessor) (api ihttp.IHTTPProcessorAPI, err error) {
	return &processorAPI{processor: httpProcessor}, err
}
