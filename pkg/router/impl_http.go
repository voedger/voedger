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
	"io"
	"log"
	"maps"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	blobprocessor "github.com/voedger/voedger/pkg/processors/blobber"
	"golang.org/x/net/netutil"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"

	"github.com/voedger/voedger/pkg/coreutils"
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
		for s.n10n.MetricNumSubcriptions() > 0 {
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

func newErrorResponder(w http.ResponseWriter) blobprocessor.ErrorResponder {
	return func(statusCode int, args ...interface{}) {
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(fmt.Sprint(args...)))
	}
}

func (s *httpService) blobHTTPRequestHandler_Write() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		appQName, wsid, headers, ok := parseURLParams(req, resp)
		if !ok {
			return
		}
		if !s.blobRequestHandler.HandleWrite(appQName, wsid, headers, req.Context(), req.URL.Query(),
			newBLOBOKResponseIniter(resp), req.Body, func(statusCode int, args ...interface{}) {
				WriteTextResponse(resp, fmt.Sprint(args...), statusCode)
			}) {
			resp.WriteHeader(http.StatusServiceUnavailable)
			resp.Header().Add("Retry-After", strconv.Itoa(DefaultRetryAfterSecondsOn503))
		}
	}
}


func (s *httpService) blobHTTPRequestHandler_Read() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		appQName, wsid, headers, ok := parseURLParams(req, resp)
		if !ok {
			return
		}
		vars := mux.Vars(req)
		existingBLOBIDOrSUID := vars[URLPlaceholder_blobIDOrSUUID]
		if !s.blobRequestHandler.HandleRead(appQName, wsid, headers, req.Context(),
			newBLOBOKResponseIniter(resp), func(statusCode int, args ...interface{}) {
				WriteTextResponse(resp, fmt.Sprint(args...), statusCode)
			}, existingBLOBIDOrSUID) {
			resp.WriteHeader(http.StatusServiceUnavailable)
			resp.Header().Add("Retry-After", strconv.Itoa(DefaultRetryAfterSecondsOn503))
		}
	}
}

func parseURLParams(req *http.Request, resp http.ResponseWriter) (appQName appdef.AppQName, wsid istructs.WSID, headers http.Header, ok bool) {
	vars := mux.Vars(req)
	wsidUint, err := strconv.ParseUint(vars[URLPlaceholder_wsid], utils.DecimalBase, utils.BitSize64)
	if err != nil {
		// notest: checked by router url rule
		panic(err)
	}
	headers = maps.Clone(req.Header)
	if _, ok := headers[coreutils.Authorization]; !ok {
		// no token among headers -> look among cookies
		// no token among cookies as well -> just do nothing, 403 will happen on call helper commands further in BLOBs processor
		cookie, err := req.Cookie(coreutils.Authorization)
		if !errors.Is(err, http.ErrNoCookie) {
			if err != nil {
				// notest
				panic(err)
			}
			val, err := url.QueryUnescape(cookie.Value)
			if err != nil {
				WriteTextResponse(resp, "failed to unescape cookie '"+cookie.Value+"'", http.StatusBadRequest)
				return appQName, wsid, headers, false
			}
			// authorization token in cookies -> q.sys.DownloadBLOBAuthnz requires it in headers
			headers[coreutils.Authorization] = []string{val}
		}
	}
	appQName = appdef.NewAppQName(vars[URLPlaceholder_appOwner], vars[URLPlaceholder_appName])
	return appQName, istructs.WSID(wsidUint), headers, true
}

func newBLOBOKResponseIniter(r http.ResponseWriter) func(headersKV ...string) io.Writer {
	return func(headersKV ...string) io.Writer {
		for i := 0; i < len(headersKV); i += 2 {
			r.Header().Set(headersKV[i], headersKV[i+1])
		}
		r.WriteHeader(http.StatusOK)
		return r
	}
}

func (s *httpService) registerHandlersV1() {
	/*
		launching app from localhost from browser. Trying to execute POST from web app within browser.
		Browser sees that hosts differs: from localhost to alpha -> need CORS -> denies POST and executes the same request with OPTIONS header
		-> need to allow OPTIONS
	*/
	if s.blobRequestHandler != nil {
		s.router.Handle(fmt.Sprintf("/blob/{%s}/{%s}/{%s:[0-9]+}", URLPlaceholder_appOwner, URLPlaceholder_appName, URLPlaceholder_wsid),
			corsHandler(s.blobHTTPRequestHandler_Write())).
			Methods("POST", "OPTIONS").
			Name("blob write")

		// allowed symbols according to see base64.URLEncoding
		s.router.Handle(fmt.Sprintf("/blob/{%s}/{%s}/{%s:[0-9]+}/{%s:[a-zA-Z0-9-_]+}", URLPlaceholder_appOwner,
			URLPlaceholder_appName, URLPlaceholder_wsid, URLPlaceholder_blobIDOrSUUID), corsHandler(s.blobHTTPRequestHandler_Read())).
			Methods("POST", "GET", "OPTIONS").
			Name("blob read")
	}
	s.router.HandleFunc(fmt.Sprintf("/api/{%s}/{%s}/{%s:[0-9]+}/{%s:[a-zA-Z0-9_/.]+}", URLPlaceholder_appOwner, URLPlaceholder_appName,
		URLPlaceholder_wsid, URLPlaceholder_resourceName), corsHandler(RequestHandler(s.bus, s.busTimeout, s.numsAppsWorkspaces))).
		Methods("POST", "PATCH", "OPTIONS").Name("api")

	s.router.Handle("/n10n/channel", corsHandler(s.subscribeAndWatchHandler())).Methods("GET")
	s.router.Handle("/n10n/subscribe", corsHandler(s.subscribeHandler())).Methods("GET")
	s.router.Handle("/n10n/unsubscribe", corsHandler(s.unSubscribeHandler())).Methods("GET")
	s.router.Handle("/n10n/update/{offset:[0-9]{1,10}}", corsHandler(s.updateHandler()))
}

func RequestHandler(bus ibus.IBus, busTimeout time.Duration, numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		queueRequest, ok := createRequest(req.Method, req, resp, numsAppsWorkspaces)
		if !ok {
			return
		}

		queueRequest.Resource = vars[URLPlaceholder_resourceName]

		// req's BaseContext is router service's context. See service.Start()
		// router app closing or client disconnected -> req.Context() is done
		// will create new cancellable context and cancel it if http section send is failed.
		// requestCtx.Done() -> SendRequest2 implementation will notify the handler that the consumer has left us
		requestCtx, cancel := context.WithCancel(req.Context())
		defer cancel() // to avoid context leak
		res, sections, secErr, err := bus.SendRequest2(requestCtx, queueRequest, busTimeout)
		if err != nil {
			logger.Error("IBus.SendRequest2 failed on ", queueRequest.Resource, ":", err, ". Body:\n", string(queueRequest.Body))
			status := http.StatusInternalServerError
			if errors.Is(err, ibus.ErrBusTimeoutExpired) {
				status = http.StatusServiceUnavailable
			}
			WriteTextResponse(resp, err.Error(), status)
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
