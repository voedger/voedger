/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttpctl

import (
	"context"
	"io/fs"

	"github.com/voedger/voedger/pkg/goutils/logger"

	"github.com/voedger/voedger/pkg/ihttp"
)

type httpProcessorController struct {
	processor          ihttp.IHTTPProcessor
	staticResources    map[string]fs.FS
	redirections       RedirectRoutes
	defaultRedirection DefaultRedirectRoute
	apps               AppRequestHandlers
}

func (hc *httpProcessorController) Prepare() (err error) {
	return nil
}

func (hc *httpProcessorController) Run(ctx context.Context) {
	for path, fs := range hc.staticResources {
		hc.processor.DeployStaticContent(path, fs)
		logger.Info(path, "deployed")
	}
	for src, dst := range hc.redirections {
		hc.processor.AddReverseProxyRoute(src, dst)
		logger.Info("redirection", src, arrow, dst, "added")
	}
	for src, dst := range hc.defaultRedirection {
		hc.processor.SetReverseProxyRouteDefault(src, dst)
		logger.Info("default redirection", src, arrow, dst, "added")
	}

	for _, appRequestHandler := range hc.apps {
		if err := hc.processor.DeployApp(appRequestHandler.AppQName, appRequestHandler.NumPartitions, appRequestHandler.NumAppWS); err != nil {
			panic(err)
		}
		for partNo, handler := range appRequestHandler.Handlers {
			if err := hc.processor.DeployAppPartition(appRequestHandler.AppQName, partNo, handler); err != nil {
				panic(err)
			}
		}
	}

	<-ctx.Done()

	for _, appRequestHandler := range hc.apps {
		for partNo := range appRequestHandler.Handlers {
			if err := hc.processor.UndeployAppPartition(appRequestHandler.AppQName, partNo); err != nil {
				panic(err)
			}
		}
		if err := hc.processor.UndeployApp(appRequestHandler.AppQName); err != nil {
			panic(err)
		}
	}
}
