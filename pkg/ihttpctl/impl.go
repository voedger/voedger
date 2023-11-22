/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttpctl

import (
	"context"
	"io/fs"

	"github.com/untillpro/goutils/logger"

	"github.com/voedger/voedger/pkg/ihttp"
)

type httpProcessorController struct {
	api                ihttp.IHTTPProcessorAPI
	staticResources    map[string]fs.FS
	redirections       RedirectRoutes
	defaultRedirection DefaultRedirectRoute
}

func (hc *httpProcessorController) Prepare() (err error) {
	return nil
}

// nolint
func (hc *httpProcessorController) Run(ctx context.Context) {
	for path, fs := range hc.staticResources {
		hc.api.DeployStaticContent(path, fs)
		logger.Info(path, "deployed")
	}
	for src, dst := range hc.redirections {
		hc.api.AddReverseProxyRoute(src, dst)
		logger.Info("redirection", src, arrow, dst, "added")
	}
	for src, dst := range hc.defaultRedirection {
		hc.api.AddReverseProxyRouteDefault(src, dst)
		logger.Info("default redirection", src, arrow, dst, "added")
	}
	<-ctx.Done()
}
