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
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

const (
	testWSID = istructs.MaxPseudoBaseWSID + 1
)

func TestBasicUsage_ApiArray(t *testing.T) {
	require := require.New(t)

	cases := []struct {
		name         string
		objs         []any
		err          error
		expectedJSON string
	}{
		{
			name:         "empty",
			objs:         nil,
			expectedJSON: `{"results":[]}`,
		},
		{
			name:         "empty + error",
			objs:         nil,
			expectedJSON: `{"results":[],"error":{"status":500,"message":"test error"}}`,
			err:          errors.New("test error"),
		},
		{
			name:         "empty + SysError",
			objs:         nil,
			expectedJSON: `{"results":[],"error":{"status":400,"message":"test error","qname":"sys.errQName","data":"more data"}}`,
			err:          coreutils.SysError{HTTPStatus: http.StatusBadRequest, QName: appdef.NewQName("sys", "errQName"), Message: "test error", Data: "more data"},
		},
		{
			name:         "one elem",
			objs:         []any{map[string]interface{}{"IntFld": 42, "StrFld": "str"}},
			expectedJSON: `{"results":[{"IntFld":42,"StrFld":"str"}]}`,
		},
		{
			name: "2 elems + error",
			objs: []any{
				map[string]interface{}{"IntFld": 42, "StrFld": "str"},
				struct {
					Fld3 int
					Fld4 string
				}{Fld3: 43, Fld4: `哇"呀呀`},
			},
			err:          coreutils.NewHTTPError(http.StatusBadRequest, errors.New("test error")),
			expectedJSON: `{"results":[{"IntFld":42,"StrFld":"str"},{"Fld3":43,"Fld4":"哇\"呀呀"}],"error":{"status":400,"message":"test error"}}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
				go func() {
					respWriter := responder.StreamJSON(http.StatusOK)
					for _, obj := range c.objs {
						require.NoError(respWriter.Write(obj))
					}
					respWriter.Close(c.err)
				}()
			}, bus.DefaultSendTimeout)
			defer tearDown(router)

			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v2/apps/test1/app1/workspaces/%d/queries/test.query", router.port(), testWSID))
			require.NoError(err)
			defer resp.Body.Close()

			expectJSONResp(t, c.expectedJSON, "", resp)
		})
	}
}

func TestBasicUsage_Respond(t *testing.T) {
	require := require.New(t)
	cases := []struct {
		name           string
		obj            any
		expectedJSON   string
		expectedString string
	}{
		{
			name:         "empty",
			expectedJSON: "{}",
		},
		{
			name:           "string",
			obj:            "test text",
			expectedString: "test text",
		},
		{
			name:           "bytes",
			obj:            []byte("test text"),
			expectedString: "test text",
		},
		{
			name: "object",
			obj: struct {
				Fld3 int
				Fld4 string
			}{Fld3: 43, Fld4: `哇"呀呀`},
			expectedJSON: `{"Fld3":43, "Fld4":"哇\"呀呀"}`,
		},
		{
			name:         "SysError",
			obj:          coreutils.SysError{HTTPStatus: http.StatusBadRequest, QName: appdef.NewQName("sys", "errQName"), Message: "test error", Data: "more data"},
			expectedJSON: `{"status":400,"message":"test error","qname":"sys.errQName","data":"more data"}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
				go func() {
					err := responder.Respond(bus.ResponseMeta{ContentType: httpu.ContentType_ApplicationJSON, StatusCode: http.StatusOK}, c.obj)
					require.NoError(err)
				}()
			}, bus.DefaultSendTimeout)
			defer tearDown(router)

			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v2/apps/test1/app1/workspaces/%d/queries/test.query", router.port(), testWSID))
			require.NoError(err)
			defer resp.Body.Close()

			expectJSONResp(t, c.expectedJSON, c.expectedString, resp)
		})
	}
}

func TestBeginResponseTimeout(t *testing.T) {
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		// bump the mock time to make timeout timer fire
		testingu.MockTime.Add(2 * time.Millisecond)
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
	expectResp(t, resp, httpu.ContentType_TextPlain, http.StatusServiceUnavailable)
}

type testObject struct {
	IntField int
	StrField string
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
			respWriter := responder.StreamJSON(http.StatusOK)
			defer respWriter.Close(nil)
			firstElemSendErrCh <- respWriter.Write(testObject{
				IntField: 42,
				StrField: "str",
			})

			// let's wait for the client close
			<-clientClosed

			// requestCtx closes not immediately after resp.Body.Close(). So let's wait for ctx close
			for requestCtx.Err() == nil {
			}

			// the request is closed -> the next section should fail with context.ContextCanceled error. Check it in the test
			expectedErrCh <- respWriter.Write(testObject{
				IntField: 43,
				StrField: "str1",
			})
		}()
	}, bus.SendTimeout(5*time.Second))
	defer tearDown(router)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v2/apps/test1/app1/workspaces/%d/queries/test.query", router.port(), testWSID))
	require.NoError(err)

	// ensure the first element is sent successfully
	require.NoError(<-firstElemSendErrCh)

	// read out the the first element
	entireResp := []byte{}
	for string(entireResp) != `{"results":[{"IntField":42,"StrField":"str"}` {
		buf := make([]byte, 512)
		n, err := resp.Body.Read(buf)
		require.NoError(err)
		entireResp = append(entireResp, buf[:n]...)
		log.Println(string(entireResp))
	}

	// close the request and signal to the handler to try to send to the disconnected client
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
	expectResp(t, resp, httpu.ContentType_TextPlain, http.StatusOK)
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
	setDisconnectOnWriteResponse := make(chan any)
	expectedErrCh := make(chan error)
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		go func() {
			// handler, on server side
			respWriter := responder.StreamJSON(http.StatusOK)
			defer respWriter.Close(nil)
			firstElemSendErrCh <- respWriter.Write(testObject{
				IntField: 42,
				StrField: "str",
			})

			// now let's wait for setting onBeforeWriteResponse hook
			<-setDisconnectOnWriteResponse

			// next send to bus should be successful because client will be disconnected on the next writeResponse()
			// this operation should trigger the client disconnect
			expectedErrCh <- respWriter.Write(testObject{
				IntField: 42,
				StrField: "str0",
			})

			// wait for the client disconnection
			<-requestCtx.Done()

			// next sending to bus must be failed because the ctx is closed
			expectedErrCh <- respWriter.Write(testObject{
				IntField: 43,
				StrField: "str1",
			})
		}()
	}, bus.SendTimeout(time.Minute)) // a minute timeout to eliminate case when client context closes longer than bus timeout on client disconnect. It could take up to few seconds
	defer tearDown(router)

	// client side
	client := &http.Client{
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	defer client.CloseIdleConnections()
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v2/apps/test1/app1/workspaces/%d/queries/test.query", router.port(), testWSID))
	require.NoError(err)

	// ensure the first element is sent successfully
	require.NoError(<-firstElemSendErrCh)

	// read out the first section
	entireResp := []byte{}
	for string(entireResp) != `{"results":[{"IntField":42,"StrField":"str"}` {
		buf := make([]byte, 512)
		n, err := resp.Body.Read(buf)
		require.NoError(err)
		entireResp = append(entireResp, buf[:n]...)
		log.Println(string(entireResp))
	}

	// force client disconnect right before write to the socket on the next writeResponse() call
	once := sync.Once{}
	onBeforeWriteResponse = func(w http.ResponseWriter) {
		once.Do(func() { resp.Body.Close() })
	}

	// signal to the handler it could try to send the next section
	// that send will be successful from bus point of view (not disconnected yet)
	// but will be failed in router (will be disconnected right on writeResponse)
	close(setDisconnectOnWriteResponse)

	// The second write may return nil or context.Canceled depending on timing:
	// - The write to bus channel succeeds (data is delivered)
	// - Router managed to call resp.Body.Close() in the hook
	// - We've got the result of `return rs.clientCtx.Err()` at implResponseWriter.Write()
	// Both outcomes are valid - what matters is the write reached the router
	secondWriteErr := <-expectedErrCh
	if secondWriteErr != nil {
		require.ErrorIs(secondWriteErr, context.Canceled)
	}

	// next sending to the bus must be failed because the requestCtx is closed
	require.ErrorIs(<-expectedErrCh, context.Canceled)

	router.expectClientDisconnection(t)

	// moved here to avoid data race on require.* failures
	onBeforeWriteResponse = nil
}

func TestAdminService(t *testing.T) {
	require := require.New(t)
	router := setUp(t, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		go bus.ReplyJSON(responder, http.StatusOK, "test resp AdminService")
	}, bus.DefaultSendTimeout)
	defer tearDown(router)

	t.Run("basic", func(t *testing.T) {
		resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/test1/app1/%d/c.somefunc_AdminService", router.adminPort(), testWSID), "application/json", http.NoBody)
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
		// hostport
		_, err = net.DialTimeout("tcp", fmt.Sprintf("%v:%d", nonLocalhostIP, router.adminPort()), 1*time.Second)
		require.Error(err)
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
	requestSender := bus.NewIRequestSender(testingu.MockTime, sendTimeout, requestHandler)
	httpSrv, acmeSrv, adminService := Provide(rp, nil, nil, nil, requestSender,
		map[appdef.AppQName]istructs.NumAppWorkspaces{istructs.AppQName_test1_app1: 10}, nil, nil, nil)
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

func expectJSONResp(t *testing.T, expectedJSON string, expectedString string, resp *http.Response) {
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	if len(expectedJSON) > 0 {
		require.JSONEq(t, expectedJSON, string(b))
	} else {
		require.Equal(t, expectedString, string(b))
	}
	expectResp(t, resp, httpu.ContentType_ApplicationJSON, http.StatusOK)
}

func expectResp(t *testing.T, resp *http.Response, contentType string, statusCode int) {
	t.Helper()
	require.Equal(t, statusCode, resp.StatusCode)
	require.Contains(t, resp.Header["Content-Type"][0], contentType, resp.Header)
	require.Equal(t, []string{"*"}, resp.Header["Access-Control-Allow-Origin"])
	require.Equal(t, []string{"Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, Blob-Name"}, resp.Header["Access-Control-Allow-Headers"])
}
