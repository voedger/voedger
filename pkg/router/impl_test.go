/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	ibus "github.com/untillpro/airs-ibus"
	"github.com/untillpro/ibusmem"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

const (
	testWSID = istructs.MaxPseudoBaseWSID + 1
)

var (
	isRouterRestartTested bool
	router                *testRouter
)

func TestBasicUsage_SingleResponse(t *testing.T) {
	require := require.New(t)
	setUp(t, func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
		bus.SendResponse(sender, ibus.Response{
			ContentType: "text/plain",
			StatusCode:  http.StatusOK,
			Data:        []byte("test resp"),
		})
	})
	defer tearDown()

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc", router.port(), testWSID), "application/json", http.NoBody)
	require.NoError(err)
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(err)
	require.Equal("test resp", string(respBodyBytes))
}

// func TestXxx(t *testing.T) {
// 	type tx struct {
// 		Str string //`json:"Str,omitempty"`
// 	}

// 	x := tx{}

// 	b, err := json.Marshal(&x)
// 	require.NoError(t, err)
// 	fmt.Println(string(b))
// }

func TestBasicUsage_SectionedResponse(t *testing.T) {
	var (
		elem1  = map[string]interface{}{"fld1": "fld1Val"}
		elem11 = map[string]interface{}{"fld2": `哇"呀呀`}
		elem21 = "e1"
		elem22 = `哇"呀呀`
		elem3  = map[string]interface{}{"total": 1}
	)
	require := require.New(t)
	setUp(t, func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
		require.Equal("test body", string(request.Body))
		require.Equal(ibus.HTTPMethodPOST, request.Method)
		require.Equal(0, request.PartitionNumber)

		require.Equal(testWSID, istructs.WSID(request.WSID))
		require.Equal("somefunc", request.Resource)
		require.Equal(0, len(request.Attachments))
		require.Equal(map[string][]string{
			"Accept-Encoding": {"gzip"},
			"Content-Length":  {"9"}, // len("test body")
			"Content-Type":    {"application/json"},
			"User-Agent":      {"Go-http-client/1.1"},
		}, request.Header)
		require.Equal(0, len(request.Query))

		// request is normally handled by processors in a separate goroutine so let's send response in a separate goroutine
		go func() {
			rs := bus.SendParallelResponse2(sender)
			require.NoError(rs.ObjectSection("obj", []string{"meta"}, elem3))
			rs.StartMapSection(`哇"呀呀Map`, []string{`哇"呀呀`, "21"})
			require.NoError(rs.SendElement("id1", elem1))
			require.NoError(rs.SendElement(`哇"呀呀2`, elem11))
			rs.StartArraySection("secArr", []string{"3"})
			require.NoError(rs.SendElement("", elem21))
			require.NoError(rs.SendElement("", elem22))
			rs.Close(nil)
		}()
	})
	defer tearDown()

	body := []byte("test body")
	bodyReader := bytes.NewReader(body)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc", router.port(), testWSID), "application/json", bodyReader)
	require.NoError(err)
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(err)

	expectedJSON := `
		{
			"sections": [
			   {
				  "elements": {
					 "total": 1
				  },
				  "path": [
					 "meta"
				  ],
				  "type": "obj"
			   },
				{
					"type": "哇\"呀呀Map",
					"path": [
						"哇\"呀呀",
						"21"
					],
					"elements": {
						"id1": {
							"fld1": "fld1Val"
						},
						"哇\"呀呀2": {
							"fld2": "哇\"呀呀"
						}
					}
				},
				{
					"type": "secArr",
					"path": [
						"3"
					],
					"elements": [
						"e1",
						"哇\"呀呀"
					]
			 	}
			]
		}`
	require.JSONEq(expectedJSON, string(respBodyBytes))
}

func TestEmptySectionedResponse(t *testing.T) {
	setUp(t, func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
		rs := bus.SendParallelResponse2(sender)
		rs.Close(nil)
	})
	defer tearDown()
	body := []byte("test body")
	bodyReader := bytes.NewReader(body)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc", router.port(), testWSID), "application/json", bodyReader)
	require.NoError(t, err)
	defer resp.Body.Close()

	expectEmptyResponse(t, resp)
}

func TestSimpleErrorSectionedResponse(t *testing.T) {
	setUp(t, func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
		rs := bus.SendParallelResponse2(sender)
		rs.Close(errors.New("test error"))
	})
	defer tearDown()

	body := []byte("")
	bodyReader := bytes.NewReader(body)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/untill/airs-bp/%d/somefunc", router.port(), testWSID), "application/json", bodyReader)
	require.Nil(t, err, err)
	defer resp.Body.Close()

	expectedJSON := `{"status":500,"errorDescription":"test error"}`

	expectJSONResp(t, expectedJSON, resp)
}

type testRouter struct {
	cancel     context.CancelFunc
	wg         *sync.WaitGroup
	httpServer pipeline.IService
	handler    func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus)
	params     RouterParams
	bus        ibus.IBus
}

func startRouter(t *testing.T, rp RouterParams, bus ibus.IBus) {
	ctx, cancel := context.WithCancel(context.Background())
	httpSrv, acmeSrv := Provide(ctx, rp, ibus.DefaultTimeout, nil, in10n.Quotas{}, nil, nil, bus, map[istructs.AppQName]istructs.AppWSAmount{istructs.AppQName_test1_app1: 10})
	require.Nil(t, acmeSrv)
	require.NoError(t, httpSrv.Prepare(nil))
	wg := sync.WaitGroup{}
	go func() {
		defer wg.Done()
		httpSrv.Run(ctx)
	}()
	router.cancel = cancel
	router.httpServer = httpSrv
}

func setUp(t *testing.T, handlerFunc func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus)) {
	if router != nil {
		router.handler = handlerFunc
		if !isRouterRestartTested {
			// let's test router restart once
			startRouter(t, router.params, router.bus)
			isRouterRestartTested = true
		}
		return
	}
	rp := RouterParams{
		Port:             0,
		WriteTimeout:     DefaultWriteTimeout,
		ReadTimeout:      DefaultReadTimeout,
		ConnectionsLimit: DefaultConnectionsLimit,
	}
	var bus ibus.IBus
	bus = ibusmem.Provide(func(requestCtx context.Context, sender interface{}, request ibus.Request) {
		router.handler(requestCtx, sender, request, bus)
	})
	router = &testRouter{
		bus:     bus,
		wg:      &sync.WaitGroup{},
		handler: handlerFunc,
	}

	startRouter(t, rp, bus)
}

func tearDown() {
	router.handler = func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
		panic("unexpected handler call")
	}
	if isRouterRestartTested {
		// let's test router shutdown once
		router.cancel()
		router.wg.Wait()
	}
}

func (t testRouter) port() int {
	return t.httpServer.(interface{ GetPort() int }).GetPort()
}

func expectEmptyResponse(t *testing.T, resp *http.Response) {
	t.Helper()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Empty(t, string(respBody))
	_, ok := resp.Header["Content-Type"]
	require.False(t, ok)
	require.Equal(t, []string{"*"}, resp.Header["Access-Control-Allow-Origin"])
	require.Equal(t, []string{"Accept, Content-Type, Content-Length, Accept-Encoding, Authorization"}, resp.Header["Access-Control-Allow-Headers"])
}

func expectJSONResp(t *testing.T, expectedJSON string, resp *http.Response) {
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, expectedJSON, string(b))
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header["Content-Type"][0], "application/json", resp.Header)
	require.Equal(t, []string{"*"}, resp.Header["Access-Control-Allow-Origin"])
	require.Equal(t, []string{"Accept, Content-Type, Content-Length, Accept-Encoding, Authorization"}, resp.Header["Access-Control-Allow-Headers"])
}
