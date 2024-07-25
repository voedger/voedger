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
	"log"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/staging/src/github.com/untillpro/ibusmem"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

const (
	testWSID = istructs.MaxPseudoBaseWSID + 1
)

var (
	isRouterStopTested   bool
	router               *testRouter
	clientDisconnections = make(chan struct{}, 1)
	previousBusTimeout   = ibus.DefaultTimeout
)

func TestBasicUsage_SingleResponse(t *testing.T) {
	require := require.New(t)
	setUp(t, func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		sender.SendResponse(ibus.Response{
			ContentType: "text/plain",
			StatusCode:  http.StatusOK,
			Data:        []byte("test resp SingleResponse"),
		})
	}, ibus.DefaultTimeout)
	defer tearDown()

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc_SingleResponse", router.port(), testWSID), "application/json", http.NoBody)
	require.NoError(err)
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(err)
	require.Equal("test resp SingleResponse", string(respBodyBytes))
}

func TestSectionedSendResponseError(t *testing.T) {
	setUp(t, func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
	}, time.Millisecond)
	defer tearDown()

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc_SectionedSendResponseError", router.port(), testWSID), "application/json", http.NoBody)
	require.NoError(t, err)
	defer resp.Body.Close()
	defer resp.Request.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, ibus.ErrBusTimeoutExpired.Error(), string(respBodyBytes))
	expectResp(t, resp, "text/plain", http.StatusServiceUnavailable)
}

func TestBasicUsage_SectionedResponse(t *testing.T) {
	var (
		elem11 = map[string]interface{}{"fld2": `哇"呀呀`}
		elem21 = "e1"
		elem22 = `哇"呀呀`
		elem3  = map[string]interface{}{"total": 1}
	)
	require := require.New(t)
	setUp(t, func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		require.Equal("test body SectionedResponse", string(request.Body))
		require.Equal(ibus.HTTPMethodPOST, request.Method)
		require.Equal(0, request.PartitionNumber)

		require.Equal(testWSID, istructs.WSID(request.WSID))
		require.Equal("somefunc_SectionedResponse", request.Resource)
		require.Empty(request.Attachments)
		require.Equal(map[string][]string{
			"Accept-Encoding": {"gzip"},
			"Content-Length":  {"27"}, // len("test body SectionedResponse")
			"Content-Type":    {"application/json"},
			"User-Agent":      {"Go-http-client/1.1"},
		}, request.Header)
		require.Empty(request.Query)

		// request is normally handled by processors in a separate goroutine so let's send response in a separate goroutine
		go func() {
			rs := sender.SendParallelResponse()
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

	body := []byte("test body SectionedResponse")
	bodyReader := bytes.NewReader(body)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/%s/%s/%d/somefunc_SectionedResponse", router.port(), AppOwner, AppName, testWSID), "application/json", bodyReader)
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
	setUp(t, func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		rs := sender.SendParallelResponse()
		rs.Close(nil)
	}, ibus.DefaultTimeout)
	defer tearDown()
	body := []byte("test body EmptySectionedResponse")
	bodyReader := bytes.NewReader(body)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc_EmptySectionedResponse", router.port(), testWSID), "application/json", bodyReader)
	require.NoError(t, err)
	defer resp.Body.Close()

	expectEmptyResponse(t, resp)
}

func TestSimpleErrorSectionedResponse(t *testing.T) {
	setUp(t, func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		rs := sender.SendParallelResponse()
		rs.Close(errors.New("test error SimpleErrorSectionedResponse"))
	}, ibus.DefaultTimeout)
	defer tearDown()

	body := []byte("")
	bodyReader := bytes.NewReader(body)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/untill/airs-bp/%d/somefunc_SimpleErrorSectionedResponse", router.port(), testWSID), "application/json", bodyReader)
	require.NoError(t, err)
	defer resp.Body.Close()

	expectedJSON := `{"status":500,"errorDescription":"test error SimpleErrorSectionedResponse"}`

	expectJSONResp(t, expectedJSON, resp)
}

func TestHandlerPanic(t *testing.T) {
	setUp(t, func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		panic("test panic HandlerPanic")
	}, ibus.DefaultTimeout)
	defer tearDown()

	body := []byte("")
	bodyReader := bytes.NewReader(body)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/untill/airs-bp/%d/somefunc_HandlerPanic", router.port(), testWSID), "application/json", bodyReader)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(respBodyBytes), "test panic HandlerPanic")
	expectResp(t, resp, "text/plain", http.StatusInternalServerError)
}

func TestClientDisconnect_CtxCanceledOnElemSend(t *testing.T) {
	require := require.New(t)
	clientClosed := make(chan struct{})
	firstElemSendErrCh := make(chan error)
	expectedErrCh := make(chan error)
	setUp(t, func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		go func() {
			rs := sender.SendParallelResponse()
			defer rs.Close(nil)
			rs.StartMapSection("secMap", []string{"2"})
			firstElemSendErrCh <- rs.SendElement("id1_ClientDisconnect_CtxCanceledOnElemSend", elem1)

			// let's wait for the client close
			<-clientClosed

			// requestCtx closes not immediately after resp.Body.Close(). So let's wait for ctx close
			for requestCtx.Err() == nil {
			}

			// the request is closed -> the next section should fail with context.ContextCanceled error. Check it in the test
			expectedErrCh <- rs.ObjectSection("objSec", []string{"3"}, 42)
		}()
	}, 5*time.Second)
	defer tearDown()

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/%s/%s/%d/somefunc_ClientDisconnect_CtxCanceledOnElemSend", router.port(), AppOwner, AppName, testWSID), "application/json", http.NoBody)
	require.NoError(err)

	// ensure the first element is sent successfully
	require.NoError(<-firstElemSendErrCh)

	// read out the the first element
	entireResp := []byte{}
	for string(entireResp) != `{"sections":[{"type":"secMap","path":["2"],"elements":{"id1_ClientDisconnect_CtxCanceledOnElemSend":{"fld1":"fld1Val"}` {
		buf := make([]byte, 512)
		n, err := resp.Body.Read(buf)
		require.NoError(err)
		entireResp = append(entireResp, buf[:n]...)
		log.Println(string(entireResp))
	}

	// close the request and signal to the handler to try to send to the disconnected client
	resp.Request.Body.Close()
	resp.Body.Close()
	close(clientClosed)

	// expect the handler got context.Canceled error on try to send to the disconnected client
	require.ErrorIs(<-expectedErrCh, context.Canceled)
	<-clientDisconnections
}

func TestCheck(t *testing.T) {
	setUp(t, func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
	}, 1*time.Second)
	defer tearDown()

	bodyReader := bytes.NewReader(nil)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/check", router.port()), "application/json", bodyReader)
	require.NoError(t, err)
	defer resp.Body.Close()
	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "ok", string(respBodyBytes))
	expectOKRespPlainText(t, resp)
}

func Test404(t *testing.T) {
	setUp(t, func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
	}, 1*time.Second)
	defer tearDown()

	bodyReader := bytes.NewReader(nil)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/wrong", router.port()), "", bodyReader)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestClientDisconnect_FailedToWriteResponse(t *testing.T) {
	require := require.New(t)
	firstElemSendErrCh := make(chan error)
	clientDisconnect := make(chan any)
	requestCtxCh := make(chan context.Context, 1)
	expectedErrCh := make(chan error)
	setUp(t, func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		go func() {
			// handler, on server side
			rs := sender.SendParallelResponse()
			defer rs.Close(nil)
			rs.StartMapSection("secMap", []string{"2"})
			firstElemSendErrCh <- rs.SendElement("id1_ClientDisconnect_FailedToWriteResponse", elem1)

			// capture the request context so that it will be able to check if it is closed indeed right before
			// write to the socket on next writeResponse() call
			requestCtxCh <- requestCtx

			// now let's wait for client disconnect
			<-clientDisconnect

			// next section should be failed on writeResponse() call because the client is disconnected
			// the expected error on this bus side is context.Canceled, on the router's side - `failed to write response`
			expectedErrCh <- rs.ObjectSection("objSec", []string{"3"}, 42)
		}()
	}, time.Hour) // one hour timeout to eliminate case when client context closes longer than bus timoeut on client disconnect. It could take up to few seconds
	defer tearDown()

	// client side
	body := []byte("")
	bodyReader := bytes.NewReader(body)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/%s/%s/%d/somefunc_ClientDisconnect_FailedToWriteResponse", router.port(), AppOwner, AppName, testWSID), "application/json", bodyReader)
	require.NoError(err)

	// ensure the first element is sent successfully
	require.NoError(<-firstElemSendErrCh)

	// read out the first section
	entireResp := []byte{}
	for string(entireResp) != `{"sections":[{"type":"secMap","path":["2"],"elements":{"id1_ClientDisconnect_FailedToWriteResponse":{"fld1":"fld1Val"}` {
		buf := make([]byte, 512)
		n, err := resp.Body.Read(buf)
		require.NoError(err)
		entireResp = append(entireResp, buf[:n]...)
		log.Println(string(entireResp))
	}

	// force client disconnect right before write to the socket on the next writeResponse() call
	once := sync.Once{}
	onBeforeWriteResponse = func(w http.ResponseWriter) {
		once.Do(func() {
			resp.Request.Body.Close()
			resp.Body.Close()

			// wait for write to the socket will be failed indeed. It happens not at once
			// that will guarantee context.Canceled error on next sending instead of possible ErrNoConsumer
			requestCtx := <-requestCtxCh
			for requestCtx.Err() == nil {
			}
		})
	}
	defer func() {
		onBeforeWriteResponse = nil
	}()

	// signal to the handler it could try to send the next section
	close(clientDisconnect)

	// ensure the next writeResponse call is failed with the expected context.Canceled error
	require.ErrorIs(<-expectedErrCh, context.Canceled)

	<-clientDisconnections
}

func TestAdminService(t *testing.T) {
	require := require.New(t)
	setUp(t, func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		sender.SendResponse(ibus.Response{
			ContentType: "text/plain",
			StatusCode:  http.StatusOK,
			Data:        []byte("test resp AdminService"),
		})
	}, ibus.DefaultTimeout)
	defer tearDown()

	t.Run("basic", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc_AdminService", router.adminPort(), testWSID), "application/json", http.NoBody)
		require.NoError(err)
		defer resp.Body.Close()

		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(err)
		require.Equal("test resp AdminService", string(respBodyBytes))
	})

	t.Run("unable to work from non-127.0.0.1", func(t *testing.T) {
		nonLocalhostIP := ""
		addrs, err := net.InterfaceAddrs()
		require.NoError(err)
		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					nonLocalhostIP = ipnet.IP.To4().String()
					break
				}
			}
		}
		if len(nonLocalhostIP) == 0 {
			t.Skip("unable to find local non-loopback ip address")
		}
		_, err = http.Post(fmt.Sprintf("http://%s:%d/api/test1/app1/%d/somefunc_AdminService2", nonLocalhostIP, router.adminPort(), testWSID), "application/json", http.NoBody)
		require.Error(err)
		log.Println(err)
	})
}

type testRouter struct {
	cancel       context.CancelFunc
	wg           *sync.WaitGroup
	httpService  pipeline.IService
	handler      func(requestCtx context.Context, sender ibus.ISender, request ibus.Request)
	bus          ibus.IBus
	adminService pipeline.IService
	busTimeout   time.Duration
}

func startRouter(t *testing.T, rp RouterParams, bus ibus.IBus, busTimeout time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	httpSrv, acmeSrv, adminService := Provide(ctx, rp, busTimeout, nil, nil, nil, bus, map[appdef.AppQName]istructs.NumAppWorkspaces{istructs.AppQName_test1_app1: 10})
	require.Nil(t, acmeSrv)
	require.NoError(t, httpSrv.Prepare(nil))
	require.NoError(t, adminService.Prepare(nil))
	router.wg.Add(2)
	go func() {
		defer router.wg.Done()
		httpSrv.Run(ctx)
	}()
	go func() {
		defer router.wg.Done()
		adminService.Run(ctx)
	}()
	router.cancel = cancel
	router.httpService = httpSrv
	router.adminService = adminService
	onRequestCtxClosed = func() {
		clientDisconnections <- struct{}{}
	}
	previousBusTimeout = busTimeout
}

func setUp(t *testing.T, handlerFunc func(requestCtx context.Context, sender ibus.ISender, request ibus.Request), busTimeout time.Duration) {
	if router != nil {
		if previousBusTimeout == busTimeout {
			router.handler = handlerFunc
			return
		}
		tearDown()
	}
	rp := RouterParams{
		Port:             0,
		WriteTimeout:     DefaultRouterWriteTimeout,
		ReadTimeout:      DefaultRouterReadTimeout,
		ConnectionsLimit: DefaultConnectionsLimit,
	}
	bus := ibusmem.Provide(func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		router.handler(requestCtx, sender, request)
	})
	router = &testRouter{
		bus:        bus,
		wg:         &sync.WaitGroup{},
		handler:    handlerFunc,
		busTimeout: busTimeout,
	}

	startRouter(t, rp, bus, busTimeout)
}

func tearDown() {
	router.handler = func(requestCtx context.Context, sender ibus.ISender, request ibus.Request) {
		panic("unexpected handler call")
	}
	select {
	case <-clientDisconnections:
		panic("unhandled client disconnection")
	default:
	}
	if !isRouterStopTested {
		// let's test router shutdown once
		router.cancel()
		router.httpService.Stop()
		router.adminService.Stop()
		router.wg.Wait()
		router = nil
		isRouterStopTested = true
	}
}

func (t testRouter) port() int {
	return t.httpService.(interface{ GetPort() int }).GetPort()
}

func (t testRouter) adminPort() int {
	return t.adminService.(interface{ GetPort() int }).GetPort()
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
