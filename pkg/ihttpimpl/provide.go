/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 */

package ihttpimpl

import (
	"net/http"
	"sync"

	"github.com/voedger/voedger/pkg/ihttp"
	coreutils "github.com/voedger/voedger/pkg/utils"
	dbcertcache "github.com/voedger/voedger/pkg/vvm/db_cert_cache"
)

func NewProcessor(params ihttp.CLIParams, routerStorage ihttp.IRouterStorage) (server ihttp.IHTTPProcessor, cleanup func(), err error) {
	r := newRouter()
	httpProcessor := httpProcessor{
		params:      params,
		router:      r,
		certCache:   dbcertcache.ProvideDbCache(routerStorage),
		acmeDomains: &sync.Map{},
		server: &http.Server{
			Addr:              coreutils.ServerAddress(params.Port),
			Handler:           r,
			ReadHeaderTimeout: defaultReadHeaderTimeout,
		},
	}
	if len(params.AcmeDomains) > 0 {
		httpProcessor.isHTTPS = true
		for _, domain := range params.AcmeDomains {
			httpProcessor.AddAcmeDomain(domain)
		}
	}
	return &httpProcessor, httpProcessor.cleanup, err
}

func NewAPI(httpProcessor ihttp.IHTTPProcessor) (api ihttp.IHTTPProcessorAPI, err error) {
	return &processorAPI{processor: httpProcessor}, err
}
