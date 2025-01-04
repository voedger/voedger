/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 */

package ihttpimpl

import (
	"net/http"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	dbcertcache "github.com/voedger/voedger/pkg/vvm/db_cert_cache"
)

func NewProcessor(params ihttp.CLIParams, routerStorage ihttp.IRouterStorage) (server ihttp.IHTTPProcessor, cleanup func()) {
	r := newRouter()
	routerAppStorage := istorage.IAppStorage(routerStorage)
	httpProcessor := &httpProcessor{
		params:      params,
		router:      r,
		certCache:   dbcertcache.ProvideDbCache(&routerAppStorage),
		acmeDomains: &sync.Map{},
		server: &http.Server{
			Addr:              coreutils.ServerAddress(params.Port),
			Handler:           r,
			ReadHeaderTimeout: defaultReadHeaderTimeout,
		},
		apps:               make(map[appdef.AppQName]*appInfo),
		numsAppsWorkspaces: make(map[appdef.AppQName]istructs.NumAppWorkspaces),
	}
	httpProcessor.requestSender = bus.NewIRequestSender(coreutils.NewITime(), bus.DefaultSendTimeout, httpProcessor.requestHandler)
	if len(params.AcmeDomains) > 0 {
		for _, domain := range params.AcmeDomains {
			httpProcessor.AddAcmeDomain(domain)
		}
	}
	return httpProcessor, httpProcessor.cleanup
}
