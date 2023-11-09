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
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"

	"github.com/untillpro/goutils/logger"

	"github.com/voedger/voedger/pkg/ihttp"
)

type httpProcessor struct {
	params                     ihttp.CLIParams
	router                     *router
	server                     *http.Server
	listener                   net.Listener
	maxNumOfConcurrentRequests int
	mu                         sync.RWMutex
	readWriteTimeout           time.Duration
	addressHandlersMap         map[addressType]*addressHandlerType
	requestContextsPool        chan *requestContextType
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
	if hs.listener, err = net.Listen("tcp", coreutils.ServerAddress(hs.params.Port)); err == nil {
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

func (hs *httpProcessor) RegisterReceiver(owner string, app string, partition int, part string, r Receiver, numOfProcessors int, bufferSize int) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	addr := addressType{owner, app, partition, part}
	if _, ok := hs.addressHandlersMap[addr]; ok {
		panic(fmt.Sprintf("receivers already exists: %+v", addr))
	}

	ctx, cancel := context.WithCancel(context.Background())
	ah := addressHandlerType{
		addr:                addr,
		httpProc:            hs,
		processorsCtx:       ctx,
		processorsCtxCancel: cancel,
		wg:                  sync.WaitGroup{},
		requestChannel:      make(requestChannelType, bufferSize),
		numOfProcessors:     numOfProcessors,
	}

	for i := 0; i < numOfProcessors; i++ {
		ah.wg.Add(1)
		proc := processor{ah: &ah, processorTimer: time.NewTimer(hs.readWriteTimeout)}
		go proc.process(r, &ah.wg, ah.requestChannel)
	}

	hs.addressHandlersMap[addr] = &ah
	logger.Info("receiver registered:", &ah)

}

// ok is false if receivers were not found
func (hs *httpProcessor) UnregisterReceiver(owner string, app string, partition int, part string) (ok bool) {
	hs.mu.RLock()
	addr := addressType{owner, app, partition, part}
	pReceiver, ok := hs.addressHandlersMap[addr]
	hs.mu.RUnlock()

	if !ok {
		logger.Info("receiver not found:", fmt.Sprintf("%+v", addr))
		return false
	}

	pReceiver.processorsCtxCancel()
	pReceiver.wg.Wait()

	hs.mu.Lock()
	delete(hs.addressHandlersMap, addr)
	hs.mu.Unlock()
	logger.Info("receiver unregistered:", pReceiver)

	return true
}

// If appropriate receivers do not exist then "BadRequestSender" should be returned
// "BadRequestSender" returns an ihttp.StatusBadRequest(400) error for every request
func (hs *httpProcessor) QuerySender(owner string, app string, partition int, part string) (sender ihttp.ISender, ok bool) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	addr := addressType{owner, app, partition, part}
	r, ok := hs.addressHandlersMap[addr]
	if ok {
		return r, ok
	}

	es := errSender{}
	_, es.status, es.err = NewResult(nil, ErrReceiverNotFound, addr.String(), "")
	return &es, ok
}

func (hs *httpProcessor) GetMetrics() (metrics Metrics) {
	metrics.MaxNumOfConcurrentRequests = hs.maxNumOfConcurrentRequests
	metrics.NumOfConcurrentRequests = hs.maxNumOfConcurrentRequests - len(hs.requestContextsPool)
	return
}

type msgDeployApp struct {
	app    istructs.AppQName
	partNo istructs.PartitionID
}

type msgDeployAppPartition struct {
	msgDeployApp
	commandHandler ihttp.ISender
	queryHandler   ihttp.ISender
}

type msgCreateSubRoute struct {
	resource string
	subRoute string
}

type msgDeployStaticContent struct {
	resource string
	fs       fs.FS
}

type processorAPI struct {
	senderHttp ihttp.ISender
}

func (api *processorAPI) DeployStaticContent(ctx context.Context, resource string, fs fs.FS) (err error) {
	msg := msgDeployStaticContent{
		resource: resource,
		fs:       fs,
	}
	_, _, err = api.senderHttp.Send(ctx, msg, NullHandler)
	return err
}

func (api *processorAPI) DeployAppPartition(ctx context.Context, app istructs.AppQName, partNo istructs.PartitionID, commandHandler, queryHandler ihttp.ISender) (err error) {
	msg := msgDeployAppPartition{
		msgDeployApp{app, partNo}, commandHandler, queryHandler,
	}
	_, _, err = api.senderHttp.Send(ctx, msg, NullHandler)
	return err
}

func (api *processorAPI) ExportApi(resource string, subRoute string) (err error) {
	msg := msgCreateSubRoute{
		resource: resource,
		subRoute: subRoute,
	}
	_, _, err = api.senderHttp.Send(context.Background(), msg, NullHandler)
	return err
}

type msgListeningPort struct {
}

func (api *processorAPI) ListeningPort(ctx context.Context) (port int, err error) {
	msg := msgListeningPort{}
	r, _, err := api.senderHttp.Send(context.Background(), msg, NullHandler)
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

func handleAppPart(commandHandler, queryHandler ihttp.ISender) http.HandlerFunc {
	_ = commandHandler // under construction
	return func(wr http.ResponseWriter, req *http.Request) {
		// <cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<wsid>/<{q,c}.funcQName>
		// got sender
		response, status, err := queryHandler.Send(context.Background(), "Data for Application", NullHandler)
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

func (hs *httpProcessor) Receiver(_ context.Context, request interface{}, _ SectionsWriterType) (response interface{}, status ihttp.Status, err error) {
	hs.router.Lock()
	defer hs.router.Unlock()
	switch v := request.(type) {
	case msgDeployAppPartition:
		// <cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<wsid>/<{q,c}.funcQName>
		route, err := hs.router.Path(
			fmt.Sprintf("/api/%s/%s/%d/q|c\\.[a-zA-Z_.]+", v.app.Owner(), v.app.Name(), v.partNo),
		)
		if err != nil {
			return NewResult(nil, err, "", "")
		}
		route.HandlerFunc(handleAppPart(v.commandHandler, v.queryHandler))
		return NewResult(nil, nil, "", "")

	case msgDeployStaticContent:
		resource := staticPath + v.resource
		f := func(wr http.ResponseWriter, req *http.Request) {
			fs := http.FileServer(http.FS(v.fs))
			http.StripPrefix(resource, fs).ServeHTTP(wr, req)
		}
		f1 := func(wr http.ResponseWriter, req *http.Request) {
			var b []byte
			if sender, ok := hs.QuerySender("owner", "app", 0, "q"); ok {
				// got sender
				response, status, err := sender.Send(context.Background(), "Data for Application", NullHandler)
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
			return NewResult(nil, err, "", "")
		}
		sub := route.subRouter()
		if route, err = sub.Path("echo"); err != nil {
			return NewResult(nil, err, "", "")
		}
		route.HandlerFunc(f1)
		if route, err = sub.Path(""); err != nil {
			return NewResult(nil, err, "", "")
		}
		route.HandlerFunc(f)
		logger.Info("new handler added for router: url -", resource)
		return NewResult(nil, nil, "", "")
	case msgListeningPort:
		return NewResult(hs.listener.Addr().(*net.TCPAddr).Port, nil, "", "")
	default:
		err = fmt.Errorf("unknown message type %T", v)
		logger.Error(err)
		return NewResult(nil, err, "", "")
	}
}

func (hs *httpProcessor) cleanup() {
	if nil != hs.listener {
		hs.listener.Close()
		hs.listener = nil
	}
	if ok := hs.UnregisterReceiver("sys", "HTTPProcessor", 0, "c"); ok {
		logger.Info("httpProcessor receiver unregistered")
		return
	}
	if ok := hs.UnregisterReceiver("owner", "app", 0, "q"); ok {
		logger.Info("echo receiver unregistered")
		return
	}
	logger.Error("httpProcessor receiver could not be unregistered")
}

func setContentType_ApplicationText(wr http.ResponseWriter) {
	wr.Header().Set(coreutils.ContentType, "application/text")
}

func NewResult(response interface{}, err error, errMsg string, errData string) (resp interface{}, status ihttp.Status, e error) {
	if err == nil {
		return response, ihttp.Status{HTTPStatus: http.StatusOK}, nil
	}

	httpStatus, ok := ErrStatuses[err]
	if !ok {
		httpStatus = http.StatusInternalServerError
	}
	status = ihttp.Status{
		HTTPStatus:   httpStatus,
		ErrorMessage: errMsg,
		ErrorData:    errData,
	}
	return response, status, err
}
