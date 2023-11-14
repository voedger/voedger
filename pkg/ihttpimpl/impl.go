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
	"sync"

	"github.com/gorilla/mux"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"

	"github.com/untillpro/goutils/logger"

	"github.com/voedger/voedger/pkg/ihttp"
)

type httpProcessor struct {
	params ihttp.CLIParams
	router *mux.Router
	sync.Map
	server   *http.Server
	listener net.Listener
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

func (p *httpProcessor) HandlerFunc(resource string, prefix bool, handlerFunc func(http.ResponseWriter, *http.Request)) {
	var r *mux.Route
	if prefix {
		r = p.router.PathPrefix(resource)
	} else {
		r = p.router.Path(resource)
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

func (api *processorAPI) DeployStaticContent(_ context.Context, resource string, fs fs.FS) (err error) {
	resource = staticPath + resource
	f := func(wr http.ResponseWriter, req *http.Request) {
		fs := http.FileServer(http.FS(fs))
		http.StripPrefix(resource, fs).ServeHTTP(wr, req)
	}
	api.processor.HandlerFunc(resource, true, f)
	return
}

func (api *processorAPI) DeployAppPartition(_ context.Context, app istructs.AppQName, partNo istructs.PartitionID, commandHandler, queryHandler ihttp.ISender) (err error) {
	// <cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<wsid>/<{q,c}.funcQName>
	path := fmt.Sprintf("/api/%s/%s/%d/q|c\\.[a-zA-Z_.]+", app.Owner(), app.Name(), partNo)
	api.processor.HandlerFunc(path, false, handleAppPart())
	return
}

func (api *processorAPI) ListeningPort(_ context.Context) (port int, err error) {
	return api.processor.ListeningPort(), nil
}

func handleAppPart() http.HandlerFunc {
	return func(wr http.ResponseWriter, req *http.Request) {
		// <cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<wsid>/<{q,c}.funcQName>
		// got sender
		_, _ = wr.Write([]byte("under construction"))
	}
}
