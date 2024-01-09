/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin (refactoring)
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
	"path"
	"regexp"
	"sync"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/exp/slices"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"

	"github.com/untillpro/goutils/logger"

	"github.com/voedger/voedger/pkg/ihttp"
)

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
}

func (p *httpProcessor) SetReverseProxyRouteDefault(srcRegExp, dstRegExp string) {
	p.router.Lock()
	defer p.router.Unlock()

	p.router.redirections[len(p.router.redirections)-1] = &redirectionRoute{
		srcRegExp:        regexp.MustCompile(srcRegExp),
		dstRegExpPattern: dstRegExp,
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

func (p *httpProcessor) DeployAppPartition(app istructs.AppQName, partNo istructs.PartitionID, commandHandler, queryHandler ihttp.ISender) {
	// <cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<wsid>/<{q,c}.funcQName>
	resourcePath := fmt.Sprintf("/api/%s/%s/%d/q|c\\.[a-zA-Z_.]+", app.Owner(), app.Name(), partNo)
	p.handlePath(resourcePath, false, handleAppPart())
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

type router struct {
	contentRouter *mux.Router
	reverseProxy  *httputil.ReverseProxy
	redirections  []*redirectionRoute // last item is always exist and if it is non-null, then it is a default route
	sync.RWMutex
}

func newRouter() *router {
	return &router{
		contentRouter: mux.NewRouter(),
		reverseProxy:  &httputil.ReverseProxy{Director: func(r *http.Request) {}},
		redirections:  make([]*redirectionRoute, 1),
	}
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.RLock()
	defer r.RUnlock()

	reqPath := req.URL.EscapedPath()
	// Clean path to canonical form and redirect.
	if p := cleanPath(reqPath); p != reqPath {
		reqURL := *req.URL
		reqURL.Path = p
		p = reqURL.String()

		w.Header().Set("Location", p)
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}
	var match mux.RouteMatch
	var handler http.Handler
	if r.Match(req, &match) {
		handler = match.Handler
		req = requestWithVars(req, match.Vars)
		req = requestWithRoute(req, match.Route)
	}

	if handler == nil && errors.Is(match.MatchErr, mux.ErrMethodMismatch) {
		handler = methodNotAllowedHandler()
	}

	if handler == nil {
		handler = http.NotFoundHandler()
	}

	handler.ServeHTTP(w, req)
}

func (r *router) Match(req *http.Request, rm *mux.RouteMatch) bool {
	return r.contentRouter.Match(req, rm) || r.matchRedirections(req, rm)
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

// cleanPath returns the canonical path for p, eliminating . and .. elements.
// Borrowed from the net/http package.
// nolint
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

// Borrowed from the mux package.
func requestWithVars(r *http.Request, vars map[string]string) *http.Request {
	ctx := context.WithValue(r.Context(), varsKey, vars)
	return r.WithContext(ctx)
}

// Borrowed from the mux package.
func requestWithRoute(r *http.Request, route *mux.Route) *http.Request {
	ctx := context.WithValue(r.Context(), routeKey, route)
	return r.WithContext(ctx)
}

// methodNotAllowed replies to the request with an HTTP status code 405.
// Borrowed from the mux package.
func methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// methodNotAllowedHandler returns a simple request handler
// that replies to each request with a status code 405.
// Borrowed from the mux package.
func methodNotAllowedHandler() http.Handler { return http.HandlerFunc(methodNotAllowed) }

func handleAppPart() http.HandlerFunc {
	return func(wr http.ResponseWriter, req *http.Request) {
		// <cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<wsid>/<{q,c}.funcQName>
		// got sender
		_, _ = wr.Write([]byte("under construction"))
	}
}
