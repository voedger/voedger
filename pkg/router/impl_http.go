/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/untillpro/goutils/logger"
	"golang.org/x/net/netutil"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func (s *httpsService) Prepare(work interface{}) error {
	if err := s.httpService.Prepare(work); err != nil {
		return err
	}

	s.server.TLSConfig = &tls.Config{GetCertificate: s.crtMgr.GetCertificate, MinVersion: tls.VersionTLS12} // VersionTLS13 is unsupported by Chargebee
	return nil
}

func (s *httpsService) Run(ctx context.Context) {
	log.Printf("Starting HTTPS server on %s\n", s.server.Addr)
	logger.Info("HTTPS server Write Timeout: ", s.server.WriteTimeout)
	logger.Info("HTTPS server Read Timeout: ", s.server.ReadTimeout)
	if err := s.server.ServeTLS(s.listener, "", ""); err != http.ErrServerClosed {
		log.Fatalf("Service.ServeTLS() failure: %s", err)
	}
}

// pipeline.IService
func (s *httpService) Prepare(work interface{}) (err error) {
	s.router = mux.NewRouter()

	// https://dev.untill.com/projects/#!627072
	s.router.SkipClean(true)

	if err = s.registerHandlers(s.busTimeout, s.appsWSAmount); err != nil {
		return err
	}

	if s.listener, err = net.Listen("tcp", coreutils.ServerAddress(s.RouterParams.Port)); err != nil {
		return err
	}

	if s.RouterParams.ConnectionsLimit > 0 {
		s.listener = netutil.LimitListener(s.listener, s.RouterParams.ConnectionsLimit)
	}

	s.server = &http.Server{
		Addr:         coreutils.ServerAddress(s.RouterParams.Port),
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.RouterParams.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.RouterParams.WriteTimeout) * time.Second,
	}

	return nil
}

// pipeline.IService
func (s *httpService) Run(ctx context.Context) {
	s.server.BaseContext = func(l net.Listener) context.Context {
		return ctx // need to track both client disconnect and app finalize
	}
	logger.Info("Starting HTTP server on", s.listener.Addr().(*net.TCPAddr).String())
	if err := s.server.Serve(s.listener); err != http.ErrServerClosed {
		log.Println("main HTTP server failure: " + err.Error())
	}
}

// pipeline.IService
func (s *httpService) Stop() {
	// ctx here is used to avoid eternal waiting for close idle connections and listeners
	// all connections and listeners are closed in the explicit way (they're tracks ctx.Done()) so it is not necessary to track ctx here
	if err := s.server.Shutdown(context.Background()); err != nil {
		log.Println("http server Shutdown() failed: " + err.Error())
		s.listener.Close()
		s.server.Close()
	}
	if s.n10n != nil {
		for s.n10n.MetricNumSubcriptions() > 0 {
			time.Sleep(subscriptionsCloseCheckInterval)
		}
	}
	s.blobWG.Wait()
}

func (s *httpService) GetPort() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

func (s *httpService) registerHandlers(busTimeout time.Duration, appsWSAmount map[istructs.AppQName]istructs.AppWSAmount) (err error) {
	redirectMatcher, err := s.getRedirectMatcher()
	if err != nil {
		return err
	}
	s.router.HandleFunc("/api/check", corsHandler(checkHandler())).Methods("POST", "OPTIONS").Name("router check")
	/*
		launching app from localhost from browser. Trying to execute POST from web app within browser.
		Browser sees that hosts differs: from localhost to alpha -> need CORS -> denies POST and executes the same request with OPTIONS header
		-> need to allow OPTIONS
	*/
	if s.BlobberParams != nil {
		s.router.Handle(fmt.Sprintf("/blob/{%s}/{%s}/{%s:[0-9]+}", appOwner, appName, wsid), corsHandler(s.blobWriteRequestHandler())).
			Methods("POST", "OPTIONS").
			Name("blob write")
		s.router.Handle(fmt.Sprintf("/blob/{%s}/{%s}/{%s:[0-9]+}/{%s:[0-9]+}", appOwner, appName, wsid, blobID), corsHandler(s.blobReadRequestHandler())).
			Methods("POST", "GET", "OPTIONS").
			Name("blob read")
	}
	s.router.HandleFunc(fmt.Sprintf("/api/{%s}/{%s}/{%s:[0-9]+}/{%s:[a-zA-Z0-9_/.]+}", appOwner, appName,
		wsid, resourceName), corsHandler(requestHandler(s.bus, busTimeout, appsWSAmount))).
		Methods("POST", "PATCH", "OPTIONS").Name("api")

	s.router.Handle("/n10n/channel", corsHandler(s.subscribeAndWatchHandler())).Methods("GET")
	s.router.Handle("/n10n/subscribe", corsHandler(s.subscribeHandler())).Methods("GET")
	s.router.Handle("/n10n/unsubscribe", corsHandler(s.unSubscribeHandler())).Methods("GET")
	s.router.Handle("/n10n/update/{offset:[0-9]{1,10}}", corsHandler(s.updateHandler()))

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

	// must be the last handler
	s.router.MatcherFunc(redirectMatcher).Name("reverse proxy")
	return nil
}

func requestHandler(bus ibus.IBus, busTimeout time.Duration, appsWSAmount map[istructs.AppQName]istructs.AppWSAmount) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		queueRequest, ok := createRequest(req.Method, req, resp, appsWSAmount)
		if !ok {
			return
		}

		queueRequest.Resource = vars[resourceName]

		// req's BaseContext is router service's context. See service.Start()
		// router app closing or client disconnected -> req.Context() is done
		// will create new cancellable context and cancel it if http section send is failed.
		// requestCtx.Done() -> SendRequest2 implementation will notify the handler that the consumer has left us
		requestCtx, cancel := context.WithCancel(req.Context())
		defer cancel() // to avoid context leak
		res, sections, secErr, err := bus.SendRequest2(requestCtx, queueRequest, busTimeout)
		if err != nil {
			logger.Error("IBus.SendRequest2 failed on ", queueRequest.Resource, ":", err, ". Body:\n", string(queueRequest.Body))
			writeTextResponse(resp, err.Error(), http.StatusInternalServerError)
			return
		}

		if sections == nil {
			resp.Header().Set(coreutils.ContentType, res.ContentType)
			resp.WriteHeader(res.StatusCode)
			writeResponse(resp, string(res.Data))
			return
		}
		writeSectionedResponse(requestCtx, resp, sections, secErr, cancel)
	}
}

func corsHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if logger.IsVerbose() {
			logger.Verbose("serving ", r.Method, " ", r.URL.Path)
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
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
