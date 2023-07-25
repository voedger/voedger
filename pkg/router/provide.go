/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"net/http"
	"time"

	ibus "github.com/untillpro/airs-ibus"
	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	"golang.org/x/crypto/acme/autocert"
)

// port == 443 -> httpsService + acmeService, otherwise -> httpService only, acmeService is nil
func Provide(hvmCtx context.Context, rp RouterParams, aBusTimeout time.Duration, broker in10n.IN10nBroker, quotas in10n.Quotas, bp *BlobberParams, autocertCache autocert.Cache,
	bus ibus.IBus, appsWSAmount map[istructs.AppQName]istructs.AppWSAmount) (httpSrv pipeline.IService, acmeSrv pipeline.IService) {
	httpService := &httpService{
		RouterParams:  rp,
		n10n:          broker,
		BlobberParams: bp,
		bus:           bus,
		busTimeout:    aBusTimeout,
		appsWSAmount:  appsWSAmount,
	}
	if bp != nil {
		bp.procBus = iprocbusmem.Provide(bp.ServiceChannels)
		for i := 0; i < bp.BLOBWorkersNum; i++ {
			httpService.blobWG.Add(1)
			go func() {
				defer httpService.blobWG.Done()
				blobMessageHandler(hvmCtx, bp.procBus.ServiceChannel(0, 0), bp.BLOBStorage, bus, aBusTimeout)
			}()
		}

	}
	if rp.Port != HTTPSPort {
		return httpService, nil
	}
	crtMgr := &autocert.Manager{
		/*
			В том случае если требуется тестировать выпуск большого количества сертификатов для разных доменов,
			то нужно использовать тестовый контур компании. Для этого в Manager требуется переопределить DirectoryURL в клиенте на
			https://acme-staging-v02.api.letsencrypt.org/directory :
			Client: &acme.Client{
				DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
			},
			поскольку есть квоты на выпуск сертификатов - на количество доменов,  сертификатов в единицу времени и пр.
		*/
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(rp.HTTP01ChallengeHosts...),
		Cache:      autocertCache,
	}
	if crtMgr.Cache == nil {
		crtMgr.Cache = autocert.DirCache(rp.CertDir)
	}
	httpsService := &httpsService{
		httpService: httpService,
		crtMgr:      crtMgr,
	}

	// handle Lets Encrypt callback over 80 port - only port 80 allowed
	acmeService := &acmeService{
		Server: http.Server{
			Addr:         ":80",
			ReadTimeout:  DefaultACMEServerReadTimeout,
			WriteTimeout: DefaultACMEServerWriteTimeout,
			Handler:      crtMgr.HTTPHandler(nil),
		},
	}
	acmeServiceHadler := crtMgr.HTTPHandler(nil)
	if logger.IsVerbose() {
		acmeService.Handler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			logger.Verbose("acme server request:", r.Method, r.Host, r.RemoteAddr, r.RequestURI, r.URL.String())
			acmeServiceHadler.ServeHTTP(rw, r)
		})
	} else {
		acmeService.Handler = acmeServiceHadler
	}
	return httpsService, acmeService
}
