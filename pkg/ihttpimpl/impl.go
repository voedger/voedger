/*
 * Copyright (c) 2022-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin (refactoring)
 */

package ihttpimpl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"sync"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"

	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/ibus"
	"github.com/voedger/voedger/pkg/ihttp"
)

type httpProcessor struct {
	params   ihttp.CLIParams
	router   *router
	server   *http.Server
	listener net.Listener
	bus      ibus.IBus
}

type router struct {
	sync.RWMutex
	routes []*route
}

type route struct {
	handler  http.HandlerFunc
	matchers []matcher
}

type matcher interface {
	match(*http.Request, *RouteMatch) bool
}

type RouteMatch struct {
	route   *route
	handler http.Handler
}

type routeRegExp regexp.Regexp

func newRRegExp(exp string) (*routeRegExp, error) {
	reg, err := regexp.Compile(exp)
	if err != nil {
		return nil, err
	}
	return (*routeRegExp)(reg), err
}

func (r *routeRegExp) match(req *http.Request, match *RouteMatch) bool {
	ok := (*regexp.Regexp)(r).MatchString(req.URL.Path)
	return ok
}

func (r *route) match(req *http.Request, match *RouteMatch) bool {
	for _, m := range r.matchers {
		if matched := m.match(req, match); !matched {
			return false
		}
	}
	if match.route == nil {
		match.route = r
	}
	if match.handler == nil {
		match.handler = r.handler
	}
	return true
}

func (r *route) HandlerFunc(f func(http.ResponseWriter, *http.Request)) *route {
	r.handler = f
	return r
}

func (r *router) PathPrefix(resource string) (*route, error) {
	return r.NewRoute().PathPrefix(resource)
}

func (r *route) PathPrefix(resource string) (*route, error) {
	pattern := bytes.NewBufferString("")
	pattern.WriteString("^(" + resource + ")")
	matcher, err := newRRegExp(pattern.String())
	if err != nil {
		return nil, err
	}
	r.matchers = append(r.matchers, matcher)
	return r, nil
}

func (r *router) Path(resource string) (*route, error) {
	return r.NewRoute().Path(resource)
}

func (r *route) Path(resource string) (*route, error) {
	pattern := bytes.NewBufferString("")
	pattern.WriteString("(.+/)?" + resource + "$")
	matcher, err := newRRegExp(pattern.String())
	if err != nil {
		return nil, err
	}
	r.matchers = append(r.matchers, matcher)
	return r, nil
}

func (hs *httpProcessor) Prepare() (err error) {
	if hs.listener, err = net.Listen("tcp", ":"+strconv.Itoa(hs.params.Port)); err == nil {
		logger.Info("listening port:", hs.listener.Addr().(*net.TCPAddr).Port)
	}
	return
}

func (hs *httpProcessor) Run(ctx context.Context) {

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("httpProcessor started:", fmt.Sprintf("%#v", hs.params))
		err := hs.server.Serve(hs.listener)
		logger.Info("httpProcessor stopped, result:", err)
	}()

	<-ctx.Done()
	if err := hs.server.Shutdown(context.Background()); err != nil {
		logger.Error("server shutdown failed", err)
		hs.listener.Close()
		hs.server.Close()
	}

	logger.Info("waiting for the httpProcessor...")
	wg.Wait()
	logger.Info("httpProcessor done")
}

type processorAPI struct {
	senderHttp ibus.ISender
}

type msgDeployApp struct {
	app    istructs.AppQName
	partNo istructs.PartitionID
}

type msgDeployAppPartition struct {
	msgDeployApp
	commandHandler ibus.ISender
	queryHandler   ibus.ISender
}

type msgCreateSubRoute struct {
	resource string
	subRoute string
}

type msgDeployStaticContent struct {
	resource string
	fs       fs.FS
}

func (api *processorAPI) DeployStaticContent(ctx context.Context, resource string, fs fs.FS) (err error) {
	msg := msgDeployStaticContent{
		resource: resource,
		fs:       fs,
	}
	_, _, err = api.senderHttp.Send(ctx, msg, ibus.NullHandler)
	return err
}

func (api *processorAPI) DeployAppPartition(ctx context.Context, app istructs.AppQName, partNo istructs.PartitionID, commandHandler, queryHandler ibus.ISender) (err error) {
	msg := msgDeployAppPartition{
		msgDeployApp{app, partNo}, commandHandler, queryHandler,
	}
	_, _, err = api.senderHttp.Send(ctx, msg, ibus.NullHandler)
	return err
}

func (api *processorAPI) ExportApi(resource string, subRoute string) (err error) {
	msg := msgCreateSubRoute{
		resource: resource,
		subRoute: subRoute,
	}
	_, _, err = api.senderHttp.Send(context.Background(), msg, ibus.NullHandler)
	return err
}

type msgListeningPort struct {
}

func (api *processorAPI) ListeningPort(ctx context.Context) (port int, err error) {
	msg := msgListeningPort{}
	r, _, err := api.senderHttp.Send(context.Background(), msg, ibus.NullHandler)
	if err != nil {
		return 0, err
	}
	return r.(int), nil
}

func (r *router) NewRoute() *route {
	route := &route{}
	r.routes = append(r.routes, route)
	return route
}

func (r *router) HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) (*route, error) {
	route, err := r.NewRoute().Path(path)
	if err != nil {
		return nil, err
	}
	return route.HandlerFunc(f), nil
}

func (r *router) match(req *http.Request, match *RouteMatch) bool {
	for _, route := range r.routes {
		if route.match(req, match) {
			return true
		}
	}
	return false
}

func (r *route) subRouter() *router {
	router := &router{}
	r.matchers = append(r.matchers, router)
	return router
}

func (r *router) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	var match RouteMatch
	if r.match(req, &match) {
		match.handler.ServeHTTP(wr, req)
		return
	}
	wr.WriteHeader(http.StatusNotFound)
	setContentType_ApplicationText(wr)
	_, _ = wr.Write([]byte("404 Not Found"))
}

func handleAppPart(commandHandler, queryHandler ibus.ISender) http.HandlerFunc {
	_ = commandHandler // under construction
	return func(wr http.ResponseWriter, req *http.Request) {
		// <cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<wsid>/<{q,c}.funcQName>
		// got sender
		response, status, err := queryHandler.Send(context.Background(), "Data for Application", ibus.NullHandler)
		if err != nil {
			wr.WriteHeader(status.HTTPStatus)
			setContentType_ApplicationText(wr)
			_, _ = wr.Write([]byte(status.ErrorMessage))

		}
		if b, err := json.Marshal(response); err == nil {
			_, _ = wr.Write(b)
		}
	}
}

func (hs *httpProcessor) Receiver(_ context.Context, request interface{}, _ ibus.SectionsWriterType) (response interface{}, status ibus.Status, err error) {
	hs.router.Lock()
	defer hs.router.Unlock()
	switch v := request.(type) {
	case msgDeployAppPartition:
		// <cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<wsid>/<{q,c}.funcQName>
		route, err := hs.router.Path(
			fmt.Sprintf("/api/%s/%s/%d/q|c\\.[a-zA-Z_.]+", v.app.Owner(), v.app.Name(), v.partNo),
		)
		if err != nil {
			return ibus.NewResult(nil, err, "", "")
		}
		route.HandlerFunc(handleAppPart(v.commandHandler, v.queryHandler))
		return ibus.NewResult(nil, nil, "", "")

	case msgDeployStaticContent:
		resource := staticPath + v.resource
		f := func(wr http.ResponseWriter, req *http.Request) {
			fs := http.FileServer(http.FS(v.fs))
			http.StripPrefix(resource, fs).ServeHTTP(wr, req)
		}
		f1 := func(wr http.ResponseWriter, req *http.Request) {
			var b []byte
			if sender, ok := hs.bus.QuerySender("owner", "app", 0, "q"); ok {
				// got sender
				response, status, err := sender.Send(context.Background(), "Data for Application", ibus.NullHandler)
				if err != nil {
					wr.WriteHeader(status.HTTPStatus)
					setContentType_ApplicationText(wr)
					_, _ = wr.Write([]byte(status.ErrorMessage))

				}
				b, err = json.Marshal(response)
				if err == nil {
					_, _ = wr.Write(b)
				}
				return
			}
			wr.WriteHeader(http.StatusNotFound)
			setContentType_ApplicationText(wr)
			_, _ = wr.Write([]byte("Not found needed query sender."))
		}
		route, err := hs.router.PathPrefix(resource)
		if err != nil {
			return ibus.NewResult(nil, err, "", "")
		}
		sub := route.subRouter()
		if route, err = sub.Path("echo"); err != nil {
			return ibus.NewResult(nil, err, "", "")
		}
		route.HandlerFunc(f1)
		if route, err = sub.Path(""); err != nil {
			return ibus.NewResult(nil, err, "", "")
		}
		route.HandlerFunc(f)
		logger.Info("new handler added for router: url -", resource)
		return ibus.NewResult(nil, nil, "", "")
	case msgListeningPort:
		return ibus.NewResult(hs.listener.Addr().(*net.TCPAddr).Port, nil, "", "")
	default:
		err = fmt.Errorf("unknown message type %T", v)
		logger.Error(err)
		return ibus.NewResult(nil, err, "", "")
	}
}

func (hs *httpProcessor) cleanup() {
	if nil != hs.listener {
		hs.listener.Close()
		hs.listener = nil
	}
	if ok := hs.bus.UnregisterReceiver("sys", "HTTPProcessor", 0, "c"); ok {
		logger.Info("httpProcessor receiver unregistered")
		return
	}
	if ok := hs.bus.UnregisterReceiver("owner", "app", 0, "q"); ok {
		logger.Info("echo receiver unregistered")
		return
	}
	logger.Error("httpProcessor receiver could not be unregistered")
}

func setContentType_ApplicationText(wr http.ResponseWriter) {
	wr.Header().Set(coreutils.ContentType, "application/text")
}
