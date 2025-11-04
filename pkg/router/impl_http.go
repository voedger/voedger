/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"golang.org/x/net/netutil"

	"github.com/voedger/voedger/pkg/istructs"
)

func (s *httpsService) Prepare(work interface{}) error {
	if err := s.httpService.Prepare(work); err != nil {
		return err
	}

	s.server.TLSConfig = &tls.Config{GetCertificate: s.crtMgr.GetCertificate, MinVersion: tls.VersionTLS12} // VersionTLS13 is unsupported by Chargebee
	return nil
}

func (s *httpsService) Run(ctx context.Context) {
	s.preRun(ctx)
	if err := s.server.ServeTLS(s.listener, "", ""); err != http.ErrServerClosed {
		s.log("ServeTLS() error: %s", err.Error())
	}
}

// pipeline.IService
func (s *httpService) Prepare(work interface{}) (err error) {
	s.router = mux.NewRouter()

	// https://dev.untill.com/projects/#!627072
	s.router.SkipClean(true)

	s.registerRouterCheckerHandler()

	s.registerHandlersV1()

	s.registerHandlersV2()

	s.registerDebugHandlers()

	// must be the last
	if err := s.registerReverseProxyHandler(); err != nil {
		return err
	}

	if s.listener, err = net.Listen("tcp", s.listenAddress); err != nil {
		return err
	}

	s.listeningPort.Store(uint32(s.listener.Addr().(*net.TCPAddr).Port)) // nolint G115

	if s.RouterParams.ConnectionsLimit > 0 {
		s.listener = netutil.LimitListener(s.listener, s.RouterParams.ConnectionsLimit)
	}

	s.server = &http.Server{
		Addr:         s.listenAddress,
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.RouterParams.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.RouterParams.WriteTimeout) * time.Second,
	}

	return nil
}

func (s *httpService) preRun(ctx context.Context) {
	s.server.BaseContext = func(l net.Listener) context.Context {
		return ctx // need to track both client disconnect and app finalize
	}
	s.log("starting on %s", s.listener.Addr().(*net.TCPAddr).String())
}

// pipeline.IService
func (s *httpService) Run(ctx context.Context) {
	s.preRun(ctx)
	if err := s.server.Serve(s.listener); err != http.ErrServerClosed {
		s.log("Serve() error: %s", err.Error())
	}
}

func (s *httpService) log(format string, args ...interface{}) {
	logger.Info(fmt.Sprintf("%s: %s", s.name, fmt.Sprintf(format, args...)))
}

// pipeline.IService
func (s *httpService) Stop() {
	// ctx here is used to avoid eternal waiting for close idle connections and listeners
	// all connections and listeners are closed in the explicit way (they're tracks ctx.Done()) so it is not necessary to track ctx here
	if err := s.server.Shutdown(context.Background()); err != nil {
		s.log("Shutdown() failed: %s", err.Error())
		s.listener.Close()
		s.server.Close()
	}
	if s.n10n != nil {
		for s.n10n.MetricNumSubscriptions() > 0 {
			time.Sleep(subscriptionsCloseCheckInterval)
		}
	}
	s.blobWG.Wait()
}

func (s *httpService) GetPort() int {
	port := s.listeningPort.Load()
	if port == 0 {
		panic("listener is not listening. Need to call http funcs before public service is started -> use IFederation.AdminFunc()")
	}
	return int(port)
}

func (s *httpService) registerDebugHandlers() {
	// pprof profile
	s.router.Handle("/debug/pprof", http.HandlerFunc(pprof.Index))
	s.router.Handle("/debug/cmdline", http.HandlerFunc(pprof.Cmdline))
	s.router.Handle("/debug/profile", http.HandlerFunc(pprof.Profile))
	s.router.Handle("/debug/symbol", http.HandlerFunc(pprof.Symbol))
	s.router.Handle("/debug/trace", http.HandlerFunc(pprof.Trace))
	s.router.Handle("/debug/{cmd}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newPath, _ := strings.CutPrefix(r.URL.Path, "/debug/")
		r.URL.Path = "/debug/pprof/" + newPath
		pprof.Index(w, r)
	})) // must be the last
}

func (s *httpService) registerReverseProxyHandler() error {
	redirectMatcher, err := s.getRedirectMatcher()
	if err != nil {
		return err
	}
	// must be the last handler
	s.router.MatcherFunc(redirectMatcher).Name("reverse proxy")
	return nil
}

func (s *httpService) registerRouterCheckerHandler() {
	s.router.HandleFunc("/api/check", corsHandler(checkHandler())).Methods("POST", "GET", "OPTIONS").Name("router check")
}

func (s *httpService) registerHandlersV1() {
	/*
		launching app from localhost from browser. Trying to execute POST from web app within browser.
		Browser sees that hosts differs: from localhost to alpha -> need CORS -> denies POST and executes the same request with OPTIONS header
		-> need to allow OPTIONS
	*/
	if s.blobRequestHandler != nil {
		s.router.Handle(fmt.Sprintf("/blob/{%s}/{%s}/{%s:[0-9]+}", URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid),
			corsHandler(s.blobHTTPRequestHandler_Write(s.numsAppsWorkspaces))).
			Methods("POST", "OPTIONS").
			Name("blob write")

		// allowed symbols according to see base64.URLEncoding
		s.router.Handle(fmt.Sprintf("/blob/{%s}/{%s}/{%s:[0-9]+}/{%s:[a-zA-Z0-9-_]+}", URLPlaceholder_appOwner,
			URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_blobIDOrSUUID), corsHandler(s.blobHTTPRequestHandler_Read(s.numsAppsWorkspaces))).
			Methods("POST", "GET", "OPTIONS").
			Name("blob read")
	}
	s.router.HandleFunc(fmt.Sprintf("/api/{%s}/{%s}/{%s:[0-9]+}/{%s:[a-zA-Z0-9_/.]+}", URLPlaceholder_appOwner, URLPlaceholder_appName,
		URLPlaceholder_wsid, URLPlaceholder_resourceName), corsHandler(RequestHandler_V1(s.requestSender, s.numsAppsWorkspaces))).
		Methods("POST", "PATCH", "OPTIONS").Name("api")

	s.router.Handle("/n10n/channel", corsHandler(s.subscribeAndWatchHandler())).Methods("GET")
	s.router.Handle("/n10n/subscribe", corsHandler(s.subscribeHandler())).Methods("GET")
	s.router.Handle("/n10n/unsubscribe", corsHandler(s.unSubscribeHandler())).Methods("GET")
	s.router.Handle("/n10n/update/{offset:[0-9]{1,10}}", corsHandler(s.updateHandler()))
}

func RequestHandler_V1(requestSender bus.IRequestSender, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return withValidateForFuncs(numsAppsWorkspaces, func(req *http.Request, rw http.ResponseWriter, data validatedData) {
		request := createBusRequest(data, req)

		// req's BaseContext is router service's context. See service.Start()
		// router app closing or client disconnected -> req.Context() is done
		// will create new cancellable context and cancel it if write to http socket is failed.
		// requestCtx.Done() -> SendRequest2 implementation will notify the handler that the consumer has left us
		requestCtx, cancel := context.WithCancel(req.Context())
		defer cancel() // to avoid context leak
		responseCh, responseMeta, responseErr, err := requestSender.SendRequest(requestCtx, request)
		if err != nil {
			logger.Error("sending request to VVM on", request.Resource, "is failed:", err, ". Body:\n", string(request.Body))
			status := http.StatusInternalServerError
			if errors.Is(err, bus.ErrSendTimeoutExpired) {
				status = http.StatusServiceUnavailable
			}
			WriteTextResponse(rw, err.Error(), status)
			return
		}

		initResponse(rw, responseMeta)
		reply_v1(requestCtx, rw, responseCh, responseErr, responseMeta.ContentType, cancel, request, responseMeta.Mode())
	})
}

func corsHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if logger.IsVerbose() {
			logger.Verbose("serving", r.Method, r.URL.Path, ", origin", r.Header.Get(httpu.Origin))
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, Blob-Name")
		if r.Method == "OPTIONS" {
			return
		}
		h.ServeHTTP(w, r)
	}
}

func checkHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		if _, err := resp.Write([]byte("ok")); err != nil {
			log.Println("failed to write 'ok' response:", err)
		}
	}
}

func initResponse(w http.ResponseWriter, responseMeta bus.ResponseMeta) {
	switch responseMeta.Mode() {
	case bus.RespondMode_StreamEvents:
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
	case bus.RespondMode_Single, bus.RespondMode_StreamJSON:
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}
	w.Header().Set(httpu.ContentType, responseMeta.ContentType)
	w.WriteHeader(responseMeta.StatusCode)
}
