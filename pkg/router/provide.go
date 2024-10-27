/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"golang.org/x/crypto/acme/autocert"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"

	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/istructs"
)

// port == 443 -> httpsService + ACMEService, otherwise -> HTTPService only, ACMEService is nil
func Provide(vvmCtx context.Context, rp RouterParams, aBusTimeout time.Duration, broker in10n.IN10nBroker, bp *BlobberParams, autocertCache autocert.Cache,
	bus ibus.IBus, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) (httpSrv IHTTPService, acmeSrv IACMEService, adminSrv IAdminService) {
	httpServ := getHttpService(vvmCtx, "HTTP server", coreutils.ServerAddress(rp.Port), rp, aBusTimeout, broker, bp, bus, numsAppsWorkspaces)

	if coreutils.IsTest() {
		adminEndpoint = "127.0.0.1:0"
	}
	adminSrv = getHttpService(vvmCtx, "Admin HTTP server", adminEndpoint, RouterParams{
		WriteTimeout:     rp.WriteTimeout,
		ReadTimeout:      rp.ReadTimeout,
		ConnectionsLimit: rp.ConnectionsLimit,
	}, aBusTimeout, broker, nil, bus, numsAppsWorkspaces)

	if rp.Port != HTTPSPort {
		return httpServ, nil, adminSrv
	}
	crtMgr := &autocert.Manager{
		/*
			If we need to test issuance of big amount of certficates for different domains then need to use test perimeter of the enterprise.
			Need to redefine DirectoryURL in Manager at:
			https://acme-staging-v02.api.letsencrypt.org/directory :
			Client: &acme.Client{
				DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
			},
			that is because there are quotas for certificate issuace: per domains amount, per amount of certificates per time etc
		*/
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(rp.HTTP01ChallengeHosts...),
		Cache:      autocertCache,
	}
	if crtMgr.Cache == nil {
		crtMgr.Cache = autocert.DirCache(rp.CertDir)
	}
	httpServ.name = "HTTPS server"
	httpsService := &httpsService{
		httpService: httpServ,
		crtMgr:      crtMgr,
	}

	// handle Lets Encrypt callback over 80 port - only port 80 allowed
	filteringLogger := log.New(&filteringWriter{log.Default().Writer()}, log.Default().Prefix(), log.Default().Flags())
	acmeService := &acmeService{
		Server: http.Server{
			Addr:         ":80",
			ReadTimeout:  DefaultACMEServerReadTimeout,
			WriteTimeout: DefaultACMEServerWriteTimeout,
			Handler:      crtMgr.HTTPHandler(nil),
			ErrorLog:     filteringLogger,
		},
	}
	return httpsService, acmeService, adminSrv
}

func getHttpService(vvmCtx context.Context, name string, listenAddress string, rp RouterParams, aBusTimeout time.Duration, broker in10n.IN10nBroker, bp *BlobberParams,
	bus ibus.IBus, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) *httpService {
	httpServ := &httpService{
		RouterParams:       rp,
		n10n:               broker,
		BlobberParams:      bp,
		bus:                bus,
		busTimeout:         aBusTimeout,
		numsAppsWorkspaces: numsAppsWorkspaces,
		listenAddress:      listenAddress,
		name:               name,
	}

	if bp != nil {
		bp.procBus = iprocbusmem.Provide(bp.ServiceChannels)
		for i := 0; i < bp.BLOBWorkersNum; i++ {
			httpServ.blobWG.Add(1)
			go func() {
				defer httpServ.blobWG.Done()
				blobMessageHandler(vvmCtx, bp.procBus.ServiceChannel(0, 0), bp.BLOBStorage, bus, aBusTimeout)
			}()
		}

	}
	return httpServ
}
