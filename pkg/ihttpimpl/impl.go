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
	"github.com/untillpro/goutils/logger"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/exp/slices"

	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/istructs"
	routerpkg "github.com/voedger/voedger/pkg/router"
	coreutils "github.com/voedger/voedger/pkg/utils"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

type appInfo struct {
	numPartitions uint
	handlers      map[istructs.PartitionID]ibus.RequestHandler
}

type httpProcessor struct {
	params       ihttp.CLIParams
	router       *router
	server       *http.Server
	listener     net.Listener
	acmeServer   *http.Server
	acmeListener net.Listener
	acmeDomains  *sync.Map
	certCache    autocert.Cache
	certManager  *autocert.Manager
	bus          ibus.IBus
	apps         map[istructs.AppQName]*appInfo
	appsWSAmount map[istructs.AppQName]istructs.AppWSAmount
	sync.RWMutex
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

	p.registerHttpHandler()

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
	p.router.Lock()
	defer p.router.Unlock()

	p.router.redirections = slices.Insert(p.router.redirections, len(p.router.redirections)-1, &redirectionRoute{
		srcRegExp:        regexp.MustCompile(srcRegExp),
		dstRegExpPattern: dstRegExp,
	})
	if p.router.reverseProxyRoute == nil {
		p.router.reverseProxyRoute = p.router.contentRouter.Name("reverse-proxy")
	}
}

func (p *httpProcessor) SetReverseProxyRouteDefault(srcRegExp, dstRegExp string) {
	p.router.Lock()
	defer p.router.Unlock()

	p.router.redirections[len(p.router.redirections)-1] = &redirectionRoute{
		srcRegExp:        regexp.MustCompile(srcRegExp),
		dstRegExpPattern: dstRegExp,
	}
	if p.router.reverseProxyRoute == nil {
		p.router.reverseProxyRoute = p.router.contentRouter.Name("reverse-proxy").MatcherFunc(p.router.matchRedirections)
	}
}

func (p *httpProcessor) AddAcmeDomain(domain string) {
	p.acmeDomains.Store(domain, struct{}{})
}

func (p *httpProcessor) DeployStaticContent(resource string, fs fs.FS) {
	resource = staticPath + resource
	f := func(wr http.ResponseWriter, req *http.Request) {
		fsHandler := http.FileServer(http.FS(fs))
		http.StripPrefix(resource, fsHandler).ServeHTTP(wr, req)
	}
	p.handlePath(resource, true, f)
}

func (p *httpProcessor) DeployAppPartition(app istructs.AppQName, partNo istructs.PartitionID, appPartitionRequestHandler ibus.RequestHandler) error {
	p.Lock()
	defer p.Unlock()

	if _, err := p.getAppPartHandler(app, partNo); !errors.Is(err, ErrAppPartitionIsNotDeployed) {
		return err
	}
	p.apps[app].handlers[partNo] = appPartitionRequestHandler
	return nil
}

func (p *httpProcessor) UndeployAppPartition(app istructs.AppQName, partNo istructs.PartitionID) error {
	p.Lock()
	defer p.Unlock()

	if _, err := p.getAppPartHandler(app, partNo); err != nil {
		return err
	}
	delete(p.apps[app].handlers, partNo)
	return nil
}

func (p *httpProcessor) getAppPartHandler(appQName istructs.AppQName, partNo istructs.PartitionID) (ibus.RequestHandler, error) {
	app, ok := p.apps[appQName]
	if !ok {
		return nil, ErrAppIsNotDeployed
	}
	if uint(partNo) >= app.numPartitions {
		return nil, ErrAppPartNoOutOfRange
	}
	handler, ok := app.handlers[partNo]
	if !ok {
		return nil, ErrAppPartitionIsNotDeployed
	}
	return handler, nil
}

func (p *httpProcessor) DeployApp(app istructs.AppQName, numPartitions uint, numAppWS uint) error {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.apps[app]; ok {
		return ErrAppAlreadyDeployed
	}
	p.apps[app] = &appInfo{
		numPartitions: numPartitions,
		handlers:      make(map[istructs.PartitionID]ibus.RequestHandler),
	}
	p.appsWSAmount[app] = istructs.AppWSAmount(numAppWS)
	return nil
}

func (p *httpProcessor) UndeployApp(app istructs.AppQName) error {
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
	delete(p.appsWSAmount, app)
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

func (p *httpProcessor) handlePath(resource string, prefix bool, handlerFunc func(http.ResponseWriter, *http.Request)) {
	p.router.Lock()
	defer p.router.Unlock()

	var r *mux.Route
	if prefix {
		r = p.router.contentRouter.PathPrefix(resource)
	} else {
		r = p.router.contentRouter.Path(resource)
	}
	r.HandlerFunc(handlerFunc)
}

func (p *httpProcessor) registerHttpHandler() {
	p.router.contentRouter.HandleFunc(
		fmt.Sprintf("/api/{%s}/{%s}/{%s:[0-9]+}/{%s:[a-zA-Z0-9_/.]+}",
			routerpkg.AppOwner,
			routerpkg.AppName,
			routerpkg.WSID,
			routerpkg.ResourceName,
		),
		corsHandler(p.httpHandler()),
	).Methods("POST", "PATCH", "OPTIONS").Name("api")
}

func (p *httpProcessor) httpHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p.RLock()
		defer p.RUnlock()

		routerpkg.RequestHandler(p.bus, busTimeout, p.appsWSAmount)(w, r)
	}
}

func (p *httpProcessor) requestHandler(ctx context.Context, sender ibus.ISender, request ibus.Request) {
	appQName, err := istructs.ParseAppQName(request.AppQName)
	if err != nil {
		coreutils.ReplyBadRequest(sender, err.Error())
		return
	}
	// TODO: is that the correct way of calculating partNo?
	partNo := istructs.PartitionID(request.WSID % partitionsAmount)
	handler, err := p.getAppPartHandler(appQName, partNo)
	if err != nil {
		coreutils.ReplyBadRequest(sender, err.Error())
		return
	}
	handler(ctx, sender, request)
}

type router struct {
	contentRouter     *mux.Router
	reverseProxy      *httputil.ReverseProxy
	redirections      []*redirectionRoute // last item is always exist and if it is non-null, then it is a default route
	reverseProxyRoute *mux.Route
	sync.RWMutex
}

func newRouter() *router {
	return &router{
		contentRouter: mux.NewRouter(),
		reverseProxy:  &httputil.ReverseProxy{Director: func(r *http.Request) {}},
		redirections:  make([]*redirectionRoute, 1),
	}
}

func (r *router) setRedirectionRoute() {
	if r.reverseProxyRoute == nil {
		r.reverseProxyRoute = r.contentRouter.Name("reverse-proxy").MatcherFunc(r.matchRedirections)
	}
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.RLock()
	defer r.RUnlock()

	r.contentRouter.ServeHTTP(w, req)
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
