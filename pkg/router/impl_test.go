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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
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
	elem1                = map[string]interface{}{"fld1": "fld1Val"}
)

func TestBasicUsage_SingleResponse(t *testing.T) {
	require := require.New(t)
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		go func() {
			bus.ReplyPlainText(responder, "test resp SingleResponse")
		}()
	}, bus.DefaultSendTimeout)
	defer tearDown(router)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc_SingleResponse", router.port(), testWSID), "application/json", http.NoBody)
	require.NoError(err)
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(err)
	require.Equal("test resp SingleResponse", string(respBodyBytes))
	expectResp(t, resp, coreutils.TextPlain, http.StatusOK)
}

func TestSectionedSendResponseError(t *testing.T) {
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		// bump the mock time to make timeout timer fire
		coreutils.MockTime.Add(2 * time.Millisecond)
		// just do not use the responder
	}, bus.SendTimeout(time.Millisecond))
	defer tearDown(router)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc_SectionedSendResponseError", router.port(), testWSID), "application/json", http.NoBody)
	require.NoError(t, err)
	defer resp.Body.Close()
	defer resp.Request.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, bus.ErrSendTimeoutExpired.Error(), string(respBodyBytes))
	expectResp(t, resp, coreutils.TextPlain, http.StatusServiceUnavailable)
}

type testObject struct {
	IntField int
	StrField string
}

func TestBasicUsage_MultiResponse(t *testing.T) {
	require := require.New(t)
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		require.Equal("test body SectionedResponse", string(request.Body))
		require.Equal(http.MethodPost, request.Method)

		require.Equal(testWSID, request.WSID)
		require.Equal("somefunc_SectionedResponse", request.Resource)
		require.Equal(map[string][]string{
			"Accept-Encoding": {"gzip"},
			"Content-Length":  {"27"}, // len("test body SectionedResponse")
			"Content-Type":    {"application/json"},
			"User-Agent":      {"Go-http-client/1.1"},
		}, request.Header)
		require.Empty(request.Query)

		// request is normally handled by processors in a separate goroutine so let's send response in a separate goroutine
		go func() {
			sender := responder.InitResponse(bus.ResponseMeta{ContentType: coreutils.ApplicationJSON, StatusCode: http.StatusOK})
			err := sender.Send(testObject{
				IntField: 42,
				StrField: `哇"呀呀`,
			})
			require.NoError(err)
			err = sender.Send(testObject{
				IntField: 50,
				StrField: `哇"呀呀2`,
			})
			require.NoError(sender.Send(nil))
			sender.Close(nil)
		}()
	}, bus.DefaultSendTimeout)
	defer tearDown(router)

	body := []byte("test body SectionedResponse")
	bodyReader := bytes.NewReader(body)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/%s/%s/%d/somefunc_SectionedResponse", router.port(), URLPlaceholder_appOwner, URLPlaceholder_appName, testWSID), "application/json", bodyReader)
	require.NoError(err)
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(err)

	expectedJSON := `{"sections":[{"type":"","elements":[
		{"IntField":42,"StrField":"哇\"呀呀"},
		{"IntField":50,"StrField":"哇\"呀呀2"},
		null
	]}]}`
	require.JSONEq(expectedJSON, string(respBodyBytes))
}

func TestEmptySectionedResponse(t *testing.T) {
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		sender := responder.InitResponse(bus.ResponseMeta{ContentType: coreutils.ApplicationJSON, StatusCode: http.StatusOK})
		sender.Close(nil)

	}, bus.DefaultSendTimeout)
	defer tearDown(router)
	body := []byte("test body EmptySectionedResponse")
	bodyReader := bytes.NewReader(body)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/somefunc_EmptySectionedResponse", router.port(), testWSID), "application/json", bodyReader)
	require.NoError(t, err)
	defer resp.Body.Close()

	expectJSONResp(t, "{}", resp)
}

func TestSimpleErrorSectionedResponse(t *testing.T) {
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		sender := responder.InitResponse(bus.ResponseMeta{ContentType: coreutils.ApplicationJSON, StatusCode: http.StatusOK})
		sender.Close(errors.New("test error SimpleErrorSectionedResponse"))
	}, bus.DefaultSendTimeout)
	defer tearDown(router)

	body := []byte("")
	bodyReader := bytes.NewReader(body)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/untill/airs-bp/%d/somefunc_SimpleErrorSectionedResponse", router.port(), testWSID), "application/json", bodyReader)
	require.NoError(t, err)
	defer resp.Body.Close()

	expectedJSON := `{"status":500,"errorDescription":"test error SimpleErrorSectionedResponse"}`

	expectJSONResp(t, expectedJSON, resp)
}

func TestHandlerPanic(t *testing.T) {
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		panic("test panic HandlerPanic")
	}, bus.DefaultSendTimeout)
	defer tearDown(router)

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
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		go func() {
			sender := responder.InitResponse(bus.ResponseMeta{ContentType: coreutils.ApplicationJSON, StatusCode: http.StatusOK})
			defer sender.Close(nil)
			firstElemSendErrCh <- sender.Send(testObject{
				IntField: 42,
				StrField: "str",
			})

			// let's wait for the client close
			<-clientClosed

			// requestCtx closes not immediately after resp.Body.Close(). So let's wait for ctx close
			for requestCtx.Err() == nil {
			}

			// the request is closed -> the next section should fail with context.ContextCanceled error. Check it in the test
			expectedErrCh <- sender.Send(testObject{
				IntField: 43,
				StrField: "str1",
			})
		}()
	}, bus.SendTimeout(5*time.Second))
	defer tearDown(router)

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/%s/%s/%d/somefunc_ClientDisconnect_CtxCanceledOnElemSend", router.port(), URLPlaceholder_appOwner, URLPlaceholder_appName, testWSID), "application/json", http.NoBody)
	require.NoError(err)

	// ensure the first element is sent successfully
	require.NoError(<-firstElemSendErrCh)

	// read out the the first element
	entireResp := []byte{}
	for string(entireResp) != `{"sections":[{"type":"","elements":[{"IntField":42,"StrField":"str"}` {
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
	router.expectClientDisconnection(t)
}

func TestCheck(t *testing.T) {
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
	}, bus.SendTimeout(1*time.Second))
	defer tearDown(router)

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
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
	}, bus.SendTimeout(1*time.Second))
	defer tearDown(router)

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
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		go func() {
			// handler, on server side
			sender := responder.InitResponse(bus.ResponseMeta{ContentType: coreutils.ApplicationJSON, StatusCode: http.StatusOK})
			defer sender.Close(nil)
			firstElemSendErrCh <- sender.Send(testObject{
				IntField: 42,
				StrField: "str",
			})

			// capture the request context so that it will be able to check if it is closed indeed right before
			// write to the socket on next writeResponse() call
			requestCtxCh <- requestCtx

			// now let's wait for client disconnect
			<-clientDisconnect

			// next elem send will be succuessful but router will fail to send in on next writeResponse() call
			expectedErrCh <- sender.Send(testObject{
				IntField: 43,
				StrField: "str1",
			})

			// next section should be failed because the client is disconnected
			expectedErrCh <- sender.Send(testObject{
				IntField: 44,
				StrField: "str2",
			})
		}()
	}, bus.SendTimeout(time.Hour)) // one hour timeout to eliminate case when client context closes longer than bus timoeut on client disconnect. It could take up to few seconds
	defer tearDown(router)

	// client side
	body := []byte("")
	bodyReader := bytes.NewReader(body)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/%s/%s/%d/somefunc_ClientDisconnect_FailedToWriteResponse", router.port(), URLPlaceholder_appOwner, URLPlaceholder_appName, testWSID), "application/json", bodyReader)
	require.NoError(err)

	// ensure the first element is sent successfully
	require.NoError(<-firstElemSendErrCh)

	// read out the first section
	entireResp := []byte{}
	for string(entireResp) != `{"sections":[{"type":"","elements":[{"IntField":42,"StrField":"str"}` {
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

	// first elem send after client disconnect should be successful, next one should fail
	require.NoError(<-expectedErrCh)

	// ensure the next writeResponse call is failed with the expected context.Canceled error
	require.ErrorIs(<-expectedErrCh, context.Canceled)

	router.expectClientDisconnection(t)
}

func TestAdminService(t *testing.T) {
	require := require.New(t)
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		go bus.ReplyPlainText(responder, "test resp AdminService")
	}, bus.DefaultSendTimeout)
	defer tearDown(router)

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
		_, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", nonLocalhostIP, router.adminPort()), 1*time.Second)
		if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "connection refused") &&
			!strings.Contains(err.Error(), "i/o timeout") {
			t.Fatal(err)
		}
		log.Println(err)
	})
}

type testRouter struct {
	cancel               context.CancelFunc
	wg                   *sync.WaitGroup
	httpService          pipeline.IService
	adminService         pipeline.IService
	sendTimeout          bus.SendTimeout
	clientDisconnections chan struct{}
}

func startRouter(t *testing.T, router *testRouter, rp RouterParams, sendTimeout bus.SendTimeout, requestHandler bus.RequestHandler) {
	ctx, cancel := context.WithCancel(context.Background())
	requestSender := bus.NewIRequestSender(coreutils.MockTime, sendTimeout, requestHandler)
	httpSrv, acmeSrv, adminService := Provide(rp, nil, nil, nil, requestSender, map[appdef.AppQName]istructs.NumAppWorkspaces{istructs.AppQName_test1_app1: 10})
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
		router.clientDisconnections <- struct{}{}
	}
}

func setUp(t *testing.T, requestHandler bus.RequestHandler, sendTimeout bus.SendTimeout) *testRouter {
	rp := RouterParams{
		Port:             0,
		WriteTimeout:     DefaultRouterWriteTimeout,
		ReadTimeout:      DefaultRouterReadTimeout,
		ConnectionsLimit: DefaultConnectionsLimit,
	}
	router := &testRouter{
		wg:                   &sync.WaitGroup{},
		sendTimeout:          sendTimeout,
		clientDisconnections: make(chan struct{}, 1),
	}

	startRouter(t, router, rp, sendTimeout, requestHandler)
	return router
}

func tearDown(router *testRouter) {
	select {
	case <-router.clientDisconnections:
		panic("unhandled client disconnection")
	default:
	}
	router.cancel()
	router.httpService.Stop()
	router.adminService.Stop()
	router.wg.Wait()
}

func (t testRouter) port() int {
	return t.httpService.(interface{ GetPort() int }).GetPort()
}

func (t testRouter) adminPort() int {
	return t.adminService.(interface{ GetPort() int }).GetPort()
}

func (t testRouter) expectClientDisconnection(tst *testing.T) {
	select {
	case <-t.clientDisconnections:
	case <-time.After(time.Second):
		tst.Fail()
	}
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
