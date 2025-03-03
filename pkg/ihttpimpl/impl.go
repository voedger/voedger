/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin (refactoring)
 * @author Alisher Nurmanov
 */

package ihttpimpl

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"sync"

	"github.com/gorilla/mux"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/exp/slices"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/istructs"
	routerpkg "github.com/voedger/voedger/pkg/router"
)

type appInfo struct {
	numPartitions istructs.NumAppPartitions
	handlers      map[istructs.PartitionID]bus.RequestHandler
}

type httpProcessor struct {
	sync.RWMutex
	params             ihttp.CLIParams
	router             *router
	server             *http.Server
	listener           net.Listener
	acmeServer         *http.Server
	acmeListener       net.Listener
	acmeDomains        *sync.Map
	certCache          autocert.Cache
	certManager        *autocert.Manager
	apps               map[appdef.AppQName]*appInfo
	numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces
	requestSender      bus.IRequestSender
}

type redirectionRoute struct {
	srcRegExp        *regexp.Regexp // if srcRegExp is null, then it is a default route
	dstRegExpPattern string
}

func (p *httpProcessor) Prepare() (err error) {
	if p.isHTTPS() {
		acmeAddr := coreutils.ServerAddress(defaultHTTPPort)
		p.certManager = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: p.hostPolicy,
			Cache:      p.certCache,
		}
		p.server.TLSConfig = &tls.Config{
			GetCertificate: p.certManager.GetCertificate,
			MinVersion:     tls.VersionTLS12, // VersionTLS13 is unsupported by Chargebee
		}
		p.acmeServer = &http.Server{
			Addr:         acmeAddr,
			ReadTimeout:  defaultACMEServerReadTimeout,
			WriteTimeout: defaultACMEServerWriteTimeout,
			Handler:      p.certManager.HTTPHandler(nil),
		}
		if p.acmeListener, err = net.Listen("tcp", acmeAddr); err == nil {
			logger.Info("listening port ", p.acmeListener.Addr().(*net.TCPAddr).Port, " for acme server")
		}
	}

	if p.listener, err = net.Listen("tcp", coreutils.ServerAddress(p.params.Port)); err == nil {
		logger.Info("listening port ", p.listener.Addr().(*net.TCPAddr).Port, " for server")
	}

	p.registerRoutes()

	return
}

func (p *httpProcessor) Run(ctx context.Context) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	isHTTPS := p.isHTTPS()
	go func() {
		defer wg.Done()
		logger.Info("httpProcessor starting:", fmt.Sprintf("%#v", p.params))
		p.server.BaseContext = func(_ net.Listener) context.Context {
			return ctx // need to track both client disconnect and app finalize
		}
		var err error
		if isHTTPS {
			err = p.server.ServeTLS(p.listener, "", "")
		} else {
			err = p.server.Serve(p.listener)
		}
		logger.Info("httpProcessor stopped, result:", err)
	}()

	if isHTTPS {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("httpProcessor's acme server starting:", fmt.Sprintf("%#v", p.params))
			err := p.acmeServer.Serve(p.acmeListener)
			logger.Info("httpProcessor's acme server  stopped, result:", err)
		}()
	}

	<-ctx.Done()
	if err := p.server.Shutdown(context.Background()); err != nil {
		logger.Error("server shutdown failed", err)
		_ = p.listener.Close()
		_ = p.server.Close()
	}

	if p.acmeServer != nil {
		if err := p.acmeServer.Shutdown(context.Background()); err != nil {
			logger.Error("acme server shutdown failed", err)
			_ = p.acmeListener.Close()
			_ = p.acmeServer.Close()
		}
	}

	logger.Info("waiting for the httpProcessor...")
	wg.Wait()
	logger.Info("httpProcessor done")
}

func (p *httpProcessor) AddReverseProxyRoute(srcRegExp, dstRegExp string) {
	p.router.addReverseProxyRoute(srcRegExp, dstRegExp)
}

func (p *httpProcessor) SetReverseProxyRouteDefault(srcRegExp, dstRegExp string) {
	p.router.setReverseProxyRouteDefault(srcRegExp, dstRegExp)
}

func (p *httpProcessor) AddAcmeDomain(domain string) {
	p.acmeDomains.Store(domain, struct{}{})
}

func (p *httpProcessor) DeployStaticContent(resource string, fs fs.FS) {
	p.router.addStaticContent(resource, fs)
}

func (p *httpProcessor) DeployAppPartition(app appdef.AppQName, partNo istructs.PartitionID, appPartitionRequestHandler bus.RequestHandler) error {
	p.Lock()
	defer p.Unlock()

	if _, err := p.getAppPartHandler(app, partNo); !errors.Is(err, ErrAppPartitionIsNotDeployed) {
		return err
	}
	p.apps[app].handlers[partNo] = appPartitionRequestHandler
	return nil
}

func (p *httpProcessor) UndeployAppPartition(app appdef.AppQName, partNo istructs.PartitionID) error {
	p.Lock()
	defer p.Unlock()

	if _, err := p.getAppPartHandler(app, partNo); err != nil {
		return err
	}
	delete(p.apps[app].handlers, partNo)
	return nil
}

func (p *httpProcessor) getAppPartHandler(appQName appdef.AppQName, partNo istructs.PartitionID) (bus.RequestHandler, error) {
	app, ok := p.apps[appQName]
	if !ok {
		return nil, ErrAppIsNotDeployed
	}
	if partNo >= istructs.PartitionID(app.numPartitions) {
		return nil, ErrAppPartNoOutOfRange
	}
	handler, ok := app.handlers[partNo]
	if !ok {
		return nil, ErrAppPartitionIsNotDeployed
	}
	return handler, nil
}

func (p *httpProcessor) DeployApp(app appdef.AppQName, numPartitions istructs.NumAppPartitions, numAppWS istructs.NumAppWorkspaces) error {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.apps[app]; ok {
		return ErrAppAlreadyDeployed
	}
	p.apps[app] = &appInfo{
		numPartitions: numPartitions,
		handlers:      make(map[istructs.PartitionID]bus.RequestHandler),
	}
	p.numsAppsWorkspaces[app] = numAppWS
	return nil
}

func (p *httpProcessor) UndeployApp(app appdef.AppQName) error {
	p.Lock()
	defer p.Unlock()

	if p.apps[app] == nil {
		return ErrAppIsNotDeployed
	}
	for _, handler := range p.apps[app].handlers {
		if handler != nil {
			return ErrActiveAppPartitionsExist
		}
	}
	delete(p.apps, app)
	delete(p.numsAppsWorkspaces, app)
	return nil
}

func (p *httpProcessor) ListeningPort() int {
	return p.listener.Addr().(*net.TCPAddr).Port
}

func (p *httpProcessor) isHTTPS() bool {
	var count int
	p.acmeDomains.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count > 0
}

func (p *httpProcessor) hostPolicy(_ context.Context, host string) error {
	if _, ok := p.acmeDomains.Load(host); !ok {
		return fmt.Errorf("acme/autocert: host %s not configured", host)
	}
	return nil
}

func (p *httpProcessor) cleanup() {
	if nil != p.listener {
		_ = p.listener.Close()
		p.listener = nil
	}
	if nil != p.acmeListener {
		_ = p.acmeListener.Close()
		p.acmeListener = nil
	}
}

func (p *httpProcessor) registerRoutes() {
	p.router.setUpRoutes(corsHandler(p.httpHandler()))
}

func (p *httpProcessor) httpHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routerpkg.RequestHandler_V1(p.requestSender, p.numsAppsWorkspaces)(w, r)
	}
}

func (p *httpProcessor) requestHandler(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
	app, ok := p.apps[request.AppQName]
	if !ok {
		bus.ReplyBadRequest(responder, ErrAppIsNotDeployed.Error())
		return
	}
	partNo := coreutils.AppPartitionID(request.WSID, app.numPartitions)
	handler, err := p.getAppPartHandler(request.AppQName, partNo)
	if err != nil {
		bus.ReplyBadRequest(responder, err.Error())
		return
	}
	handler(requestCtx, request, responder)
}

type router struct {
	router        *mux.Router
	reverseProxy  *httputil.ReverseProxy
	staticContent map[string]http.HandlerFunc
	redirections  []*redirectionRoute // last item is always exist and if it is non-null, then it is a default route
	sync.RWMutex
}

func newRouter() *router {
	return &router{
		router:        mux.NewRouter(),
		staticContent: make(map[string]http.HandlerFunc),
		reverseProxy:  &httputil.ReverseProxy{Director: func(r *http.Request) {}},
		redirections:  make([]*redirectionRoute, 1),
	}
}

func (r *router) setUpRoutes(appRequestHandler http.HandlerFunc) {
	appRequestPath := fmt.Sprintf("/api/{%s}/{%s}/{%s:[0-9]+}/{%s:[a-zA-Z0-9_/.]+}",
		routerpkg.URLPlaceholder_appOwner,
		routerpkg.URLPlaceholder_appName,
		routerpkg.URLPlaceholder_wsid,
		routerpkg.URLPlaceholder_resourceName,
	)
	r.router.HandleFunc(appRequestPath, appRequestHandler).Name("api").Methods("POST", "PATCH", "OPTIONS")
	r.router.Name("static").PathPrefix(staticPath).MatcherFunc(r.matchStaticContent)
	// set reverse proxy route last
	r.router.Name("reverse-proxy").MatcherFunc(r.matchRedirections)
}

func (r *router) addStaticContent(resource string, fs fs.FS) {
	r.Lock()
	defer r.Unlock()

	r.staticContent[resource] = staticContentHandler(staticPath+resource, fs)
}

func (r *router) addReverseProxyRoute(srcRegExp, dstRegExp string) {
	r.Lock()
	defer r.Unlock()

	r.redirections = slices.Insert(r.redirections, len(r.redirections)-1, &redirectionRoute{
		srcRegExp:        regexp.MustCompile(srcRegExp),
		dstRegExpPattern: dstRegExp,
	})
}

func (r *router) setReverseProxyRouteDefault(srcRegExp, dstRegExp string) {
	r.Lock()
	defer r.Unlock()

	r.redirections[len(r.redirections)-1] = &redirectionRoute{
		srcRegExp:        regexp.MustCompile(srcRegExp),
		dstRegExpPattern: dstRegExp,
	}
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.RLock()
	defer r.RUnlock()

	r.router.ServeHTTP(w, req)
}

func staticContentHandler(resource string, fs fs.FS) http.HandlerFunc {
	return func(wr http.ResponseWriter, req *http.Request) {
		fsHandler := http.FileServer(http.FS(fs))
		http.StripPrefix(resource, fsHandler).ServeHTTP(wr, req)
	}
}

func (r *router) matchStaticContent(req *http.Request, rm *mux.RouteMatch) (matched bool) {
	requestedURL := getFullRequestedURL(req)
	for path, handler := range r.staticContent {
		if regexp.MustCompile(path).MatchString(requestedURL) {
			rm.Route = r.router.Get("static")
			rm.Handler = handler
			return true
		}
	}
	return
}

func (r *router) matchRedirections(req *http.Request, rm *mux.RouteMatch) (matched bool) {
	for _, redirection := range r.redirections {
		if checkRedirection(redirection, r.reverseProxy, req, rm) {
			return true
		}
	}
	return
}

func checkRedirection(redirection *redirectionRoute, reverseProxy *httputil.ReverseProxy, req *http.Request, rm *mux.RouteMatch) bool {
	if redirection == nil {
		return false
	}
	requestedURL := getFullRequestedURL(req)
	if redirection.srcRegExp.MatchString(requestedURL) {
		redirectRequest(redirection, req, requestedURL)
		rm.Handler = reverseProxy
		return true
	}
	return false
}

func redirectRequest(redirection *redirectionRoute, req *http.Request, requestedURL string) {
	target := redirection.srcRegExp.ReplaceAllString(requestedURL, redirection.dstRegExpPattern)
	targetURL, err := url.Parse(target)
	if err != nil {
		panic(err)
	}
	if logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("reverse proxy: incoming %s %s%s, redirecting to %s%s", req.Method, req.Host, req.URL, targetURL.Host, targetURL.Path))
	}
	req.URL.Path = targetURL.Path
	req.Host = targetURL.Host
	req.URL.Scheme = targetURL.Scheme
	req.URL.Host = targetURL.Host
	targetQuery := targetURL.RawQuery
	if targetQuery == "" || req.URL.RawQuery == "" {
		req.URL.RawQuery = targetQuery + req.URL.RawQuery
	} else {
		req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
	}
}

func getFullRequestedURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)
	if r.URL.RawQuery != "" {
		fullURL += "?" + r.URL.RawQuery
	}
	return fullURL
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
