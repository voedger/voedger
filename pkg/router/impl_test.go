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
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"testing"
	"time"

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
	}, ibus.DefaultTimeout)
	defer tearDown()

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc", router.port(), testWSID), "application/json", http.NoBody)
	require.NoError(err)
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(err)
	require.Equal("test resp", string(respBodyBytes))
}

func TestSectionedSendResponseError(t *testing.T) {
	setUp(t, func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
	}, time.Millisecond)
	defer tearDown()

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc", router.port(), testWSID), "application/json", http.NoBody)
	require.Nil(t, err, err)
	defer resp.Body.Close()
	defer resp.Request.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	require.Equal(t, ibus.ErrTimeoutExpired.Error(), string(respBodyBytes))
	expect500RespPlainText(t, resp)
}

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
	}, ibus.DefaultTimeout)
	defer tearDown()

	body := []byte("test body")
	bodyReader := bytes.NewReader(body)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/%s/%s/%d/somefunc", router.port(), appOwner, appName, testWSID), "application/json", bodyReader)
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
	}, ibus.DefaultTimeout)
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
	}, ibus.DefaultTimeout)
	defer tearDown()

	body := []byte("")
	bodyReader := bytes.NewReader(body)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/untill/airs-bp/%d/somefunc", router.port(), testWSID), "application/json", bodyReader)
	require.Nil(t, err, err)
	defer resp.Body.Close()

	expectedJSON := `{"status":500,"errorDescription":"test error"}`

	expectJSONResp(t, expectedJSON, resp)
}

func TestHandlerPanic(t *testing.T) {
	setUp(t, func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
		panic("test panic")
	}, ibus.DefaultTimeout)
	defer tearDown()

	body := []byte("")
	bodyReader := bytes.NewReader(body)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/untill/airs-bp/%d/somefunc", router.port(), testWSID), "application/json", bodyReader)
	require.Nil(t, err)
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	require.Contains(t, string(respBodyBytes), "test panic")
	expect500RespPlainText(t, resp)
}

func TestClientDisconnectDuringSections(t *testing.T) {
	var (
		elem1 = map[string]interface{}{"fld1": "fld1Val"}
	)
	ch := make(chan struct{})
	setUp(t, func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
		go func() {
			rs := bus.SendParallelResponse2(sender)
			rs.StartMapSection("secMap", []string{"2"})
			require.Nil(t, rs.SendElement("id1", elem1))
			<-ch
			_ = rs.ObjectSection("objSec", []string{"3"}, 42)
			rs.Close(nil)
			ch <- struct{}{}
		}()
	}, 10*time.Millisecond)
	defer tearDown()

	body := []byte("")
	bodyReader := bytes.NewReader(body)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/%s/%s/%d/somefunc", router.port(), appOwner, appName, testWSID), "application/json", bodyReader)
	require.NoError(t, err)
	entireResp := []byte{}
	for string(entireResp) != `{"sections":[{"type":"secMap","path":["2"],"elements":{"id1":{"fld1":"fld1Val"}` {
		buf := make([]byte, 512)
		n, _ := resp.Body.Read(buf)
		require.Nil(t, err)
		entireResp = append(entireResp, buf[:n]...)
		log.Println(string(entireResp))
	}
	//resp.Request.Body.Close()
	resp.Body.Close()
	ch <- struct{}{}
	<-ch
}

func TestCheck(t *testing.T) {
	setUp(t, func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
	}, 1*time.Second)
	defer tearDown()

	bodyReader := bytes.NewReader(nil)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/check", router.port()), "application/json", bodyReader)
	require.Nil(t, err, err)
	defer resp.Body.Close()
	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	require.Equal(t, "ok", string(respBodyBytes))
	expectOKRespPlainText(t, resp)
}

func Test404(t *testing.T) {
	setUp(t, func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
	}, 1*time.Second)
	defer tearDown()

	bodyReader := bytes.NewReader(nil)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/wrong", router.port()), "", bodyReader)
	require.Nil(t, err, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestFailedToWriteResponse(t *testing.T) {
	var (
		elem1 = map[string]interface{}{"fld1": "fld1Val"}
	)
	ch := make(chan struct{})
	failToWrite := make(chan struct{})
	setUp(t, func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
		go func() {
			rs := bus.SendParallelResponse2(sender)
			rs.StartMapSection("secMap", []string{"2"})
			require.Nil(t, rs.SendElement("id1", elem1))

			// now let's wait for client disconnect
			ch <- struct{}{}
			<-ch

			// next section should be failed with ErrNoConsumer
			_ = rs.ObjectSection("objSec", []string{"3"}, 42)
			//require.ErrorIs(t, err, ibus.ErrNoConsumer)
			rs.Close(nil)
			ch <- struct{}{}
		}()
	}, 10*time.Second)
	onRequestCtxClosed = func() {
		failToWrite <- struct{}{}
	}
	defer tearDown()

	body := []byte("")
	bodyReader := bytes.NewReader(body)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/%s/%s/%d/somefunc", router.port(), appOwner, appName, testWSID), "application/json", bodyReader)
	require.Nil(t, err, err)
	<-ch
	onAfterSectionWrite = func(w http.ResponseWriter) {
		// disconnect the client
		resp.Body.Close()
		// wait for the write to the closed socket error. Sometimes does not appear on first write after socket close
		for {
			_, err := w.Write([]byte{0})
			if err != nil {
				break
			}
		}
		log.Println()
	}
	ch <- struct{}{}
	resp.Request.Body.Close()
	resp.Body.Close()

	// wait for fail to write response
	<-failToWrite

	// wait for communication done
	<-ch
}

type testRouter struct {
	cancel     context.CancelFunc
	wg         *sync.WaitGroup
	httpServer pipeline.IService
	handler    func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus)
	params     RouterParams
	bus        ibus.IBus
}

func startRouter(t *testing.T, rp RouterParams, bus ibus.IBus, busTimeout time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	httpSrv, acmeSrv := Provide(ctx, rp, busTimeout, nil, in10n.Quotas{}, nil, nil, bus, map[istructs.AppQName]istructs.AppWSAmount{istructs.AppQName_test1_app1: 10})
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

func setUp(t *testing.T, handlerFunc func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus), busTimeout time.Duration) {
	if router != nil {
		router.handler = handlerFunc
		if !isRouterRestartTested {
			// let's test router restart once
			startRouter(t, router.params, router.bus, busTimeout)
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

	startRouter(t, rp, bus, busTimeout)
}

func tearDown() {
	router.handler = func(requestCtx context.Context, sender interface{}, request ibus.Request, bus ibus.IBus) {
		panic("unexpected handler call")
	}
	if isRouterRestartTested {
		// let's test router shutdown once
		router.cancel()
		router.wg.Wait()
		router = nil
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

func expect500RespPlainText(t *testing.T, resp *http.Response) {
	t.Helper()
	expectResp(t, resp, "text/plain", http.StatusInternalServerError)
}

func expectOKRespPlainText(t *testing.T, resp *http.Response) {
	t.Helper()
	expectResp(t, resp, "text/plain", http.StatusOK)
}

func expectResp(t *testing.T, resp *http.Response, contentType string, statusCode int) {
	t.Helper()
	require.Equal(t, statusCode, resp.StatusCode)
	require.Contains(t, resp.Header["Content-Type"][0], contentType, resp.Header)
	require.Equal(t, []string{"*"}, resp.Header["Access-Control-Allow-Origin"])
	require.Equal(t, []string{"Accept, Content-Type, Content-Length, Accept-Encoding, Authorization"}, resp.Header["Access-Control-Allow-Headers"])
}
