/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin (refactoring)
 */

package ihttpimpl

import (
	"context"
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
	"golang.org/x/exp/slices"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"

	"github.com/untillpro/goutils/logger"

	"github.com/voedger/voedger/pkg/ihttp"
)

type httpProcessor struct {
	params   ihttp.CLIParams
	router   *router
	server   *http.Server
	listener net.Listener
}

type redirectionRoute struct {
	srcRegExp        *regexp.Regexp // if srcRegExp is null, then it is a default route
	dstRegExpPattern string
}

func (p *httpProcessor) Prepare() (err error) {
	if p.listener, err = net.Listen("tcp", coreutils.ServerAddress(p.params.Port)); err == nil {
		logger.Info("listening port:", p.listener.Addr().(*net.TCPAddr).Port)
	}
	return
}

func (p *httpProcessor) Run(ctx context.Context) {

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("httpProcessor started:", fmt.Sprintf("%#v", p.params))
		err := p.server.Serve(p.listener)
		logger.Info("httpProcessor stopped, result:", err)
	}()

	<-ctx.Done()
	if err := p.server.Shutdown(context.Background()); err != nil {
		logger.Error("server shutdown failed", err)
		p.listener.Close()
		p.server.Close()
	}

	logger.Info("waiting for the httpProcessor...")
	wg.Wait()
	logger.Info("httpProcessor done")
}

func (p *httpProcessor) AddReverseProxyRoute(srcRegExp, dstRegExp string) {
	// TODO: concurrency safety can be added via sync.RWMutex
	p.router.redirections = slices.Insert(p.router.redirections, len(p.router.redirections)-1, &redirectionRoute{
		srcRegExp:        regexp.MustCompile(srcRegExp),
		dstRegExpPattern: dstRegExp,
	})
}

func (p *httpProcessor) AddReverseProxyRouteDefault(srcRegExp, dstRegExp string) {
	// TODO: concurrency safety can be added via sync.RWMutex
	p.router.redirections[len(p.router.redirections)-1] = &redirectionRoute{
		srcRegExp:        regexp.MustCompile(srcRegExp),
		dstRegExpPattern: dstRegExp,
	}
}

func (p *httpProcessor) HandlePath(resource string, prefix bool, handlerFunc func(http.ResponseWriter, *http.Request)) {
	// TODO: concurrency safety can be added via sync.RWMutex
	var r *mux.Route
	if prefix {
		r = p.router.contentRouter.PathPrefix(resource)
	} else {
		r = p.router.contentRouter.Path(resource)
	}
	r.HandlerFunc(handlerFunc)
}

func (p *httpProcessor) ListeningPort() int {
	return p.listener.Addr().(*net.TCPAddr).Port
}

func (p *httpProcessor) cleanup() {
	if nil != p.listener {
		p.listener.Close()
		p.listener = nil
	}
}

type processorAPI struct {
	processor ihttp.IHTTPProcessor
}

func (api *processorAPI) DeployStaticContent(resource string, fs fs.FS) {
	resource = staticPath + resource
	f := func(wr http.ResponseWriter, req *http.Request) {
		fsHandler := http.FileServer(http.FS(fs))
		http.StripPrefix(resource, fsHandler).ServeHTTP(wr, req)
	}
	api.processor.HandlePath(resource, true, f)
}

func (api *processorAPI) DeployAppPartition(app istructs.AppQName, partNo istructs.PartitionID, commandHandler, queryHandler ihttp.ISender) {
	// <cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<wsid>/<{q,c}.funcQName>
	path := fmt.Sprintf("/api/%s/%s/%d/q|c\\.[a-zA-Z_.]+", app.Owner(), app.Name(), partNo)
	api.processor.HandlePath(path, false, handleAppPart())
}

func (api *processorAPI) AddReverseProxyRoute(srcRegExp, dstRegExp string) {
	api.processor.AddReverseProxyRoute(srcRegExp, dstRegExp)
}

func (api *processorAPI) AddReverseProxyRouteDefault(srcRegExp, dstRegExp string) {
	api.processor.AddReverseProxyRouteDefault(srcRegExp, dstRegExp)
}

type router struct {
	contentRouter *mux.Router
	reverseProxy  *httputil.ReverseProxy
	redirections  []*redirectionRoute // last item is always exist and if it is non-null, then it is a default route
}

func newRouter() *router {
	return &router{
		contentRouter: mux.NewRouter(),
		reverseProxy:  &httputil.ReverseProxy{Director: func(r *http.Request) {}},
		redirections:  make([]*redirectionRoute, 1),
	}
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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

	if handler == nil && match.MatchErr == mux.ErrMethodMismatch {
		handler = methodNotAllowedHandler()
	}

	if handler == nil {
		handler = http.NotFoundHandler()
	}

	handler.ServeHTTP(w, req)
}

func (r *router) Match(req *http.Request, rm *mux.RouteMatch) bool {
	// TODO: concurrency safety can be added via sync.RWMutex
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
	if p[len(p)-1] != '/' {
		p += "/"
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
func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
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
