/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"log"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
	blobprocessor "github.com/voedger/voedger/pkg/processors/blobber"
	"golang.org/x/crypto/acme/autocert"

	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
)

// port == 443 -> httpsService + ACMEService, otherwise -> HTTPService only, ACMEService is nil
// where is VVM RequestHandler? bus.RequestHandler
func Provide(rp RouterParams, broker in10n.IN10nBroker, blobRequestHandler blobprocessor.IRequestHandler, autocertCache autocert.Cache,
	requestSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) (httpSrv IHTTPService, acmeSrv IACMEService, adminSrv IAdminService) {
	httpServ := getHttpService("HTTP server", coreutils.ServerAddress(rp.Port), rp, broker, blobRequestHandler,
		requestSender, numsAppsWorkspaces)

	if coreutils.IsTest() {
		adminEndpoint = "127.0.0.1:0"
	}
	adminSrv = getHttpService("Admin HTTP server", adminEndpoint, RouterParams{
		WriteTimeout:     rp.WriteTimeout,
		ReadTimeout:      rp.ReadTimeout,
		ConnectionsLimit: rp.ConnectionsLimit,
	}, broker, nil, requestSender, numsAppsWorkspaces)

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

func getHttpService(name string, listenAddress string, rp RouterParams, broker in10n.IN10nBroker,
	blobRequestHandler blobprocessor.IRequestHandler, requestSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) *httpService {
	httpServ := &httpService{
		RouterParams:       rp,
		n10n:               broker,
		requestSender:      requestSender,
		numsAppsWorkspaces: numsAppsWorkspaces,
		listenAddress:      listenAddress,
		name:               name,
		blobRequestHandler: blobRequestHandler,
	}

	return httpServ
}
