/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

func TestNewHTTPError(t *testing.T) {
	require := require.New(t)
	t.Run("simple", func(t *testing.T) {
		sysErr := NewHTTPError(http.StatusInternalServerError, errors.New("test error"))
		require.Empty(sysErr.Data)
		require.Equal(http.StatusInternalServerError, sysErr.HTTPStatus)
		require.Equal("test error", sysErr.Message)
		require.Equal(appdef.NullQName, sysErr.QName)
		require.Equal(`{"sys.Error":{"HTTPStatus":500,"Message":"test error"}}`, sysErr.ToJSON())
	})

	t.Run("formatted", func(t *testing.T) {
		sysErr := NewHTTPErrorf(http.StatusInternalServerError, "test ", "error")
		require.Empty(sysErr.Data)
		require.Equal(http.StatusInternalServerError, sysErr.HTTPStatus)
		require.Equal("test error", sysErr.Message)
		require.Equal(appdef.NullQName, sysErr.QName)
		require.Equal(`{"sys.Error":{"HTTPStatus":500,"Message":"test error"}}`, sysErr.ToJSON())
	})
}

func TestReply(t *testing.T) {
	require := require.New(t)

	type expected struct {
		code  int
		error string
	}

	t.Run("reply errors", func(t *testing.T) {
		err := errors.New("test error")
		cases := []struct {
			desc     string
			f        func(responder IResponder)
			expected expected
		}{
			{
				desc: "ReplyErr",
				f:    func(responder IResponder) { ReplyErr(responder, err) },
				expected: expected{
					code:  http.StatusInternalServerError,
					error: `{"sys.Error":{"HTTPStatus":500,"Message":"test error"}}`,
				},
			},
			{
				desc: "ReplyErrf",
				f:    func(responder IResponder) { ReplyErrf(responder, http.StatusAccepted, "test ", "message") },
				expected: expected{
					code:  http.StatusAccepted,
					error: `{"sys.Error":{"HTTPStatus":202,"Message":"test message"}}`,
				},
			},
			{
				desc: "ReplyErrorDef",
				f:    func(responder IResponder) { ReplyErrDef(responder, err, http.StatusAccepted) },
				expected: expected{
					code:  http.StatusAccepted,
					error: `{"sys.Error":{"HTTPStatus":202,"Message":"test error"}}`,
				},
			},
			{
				desc: "SysError",
				f: func(responder IResponder) {
					err := SysError{
						HTTPStatus: http.StatusAlreadyReported,
						Message:    "test error",
						Data:       "dddfd",
						QName:      appdef.NewQName("my", "qname"),
					}
					ReplyErrDef(responder, err, http.StatusAccepted)
				},
				expected: expected{
					code:  http.StatusAlreadyReported,
					error: `{"sys.Error":{"HTTPStatus":208,"Message":"test error","QName":"my.qname","Data":"dddfd"}}`,
				},
			},
			{
				desc: "ReplyInternalServerError",
				f: func(responder IResponder) {
					ReplyInternalServerError(responder, "test", err)
				},
				expected: expected{
					code:  http.StatusInternalServerError,
					error: `{"sys.Error":{"HTTPStatus":500,"Message":"test: test error"}}`,
				},
			},
		}

		for _, c := range cases {
			t.Run(c.desc, func(t *testing.T) {
				requestSender := NewIRequestSender(MockTime, SendTimeout(GetTestBusTimeout()), func(requestCtx context.Context, request ibus.Request, responder IResponder) {
					c.f(responder)
				})
				cmdRespMeta, cmdResp, err := GetCommandResponse(context.Background(), requestSender, ibus.Request{})
				require.NoError(err)
				require.Equal(ApplicationJSON, cmdRespMeta.ContentType)
				require.Equal(c.expected.code, cmdRespMeta.StatusCode)
				require.Equal(c.expected.error, cmdResp.SysError.ToJSON())
			})
		}

	})

	t.Run("http status helpers", func(t *testing.T) {
		cases := []struct {
			statusCode      int
			f               func(responder IResponder, message string)
			expectedMessage string
		}{
			{f: ReplyUnauthorized, statusCode: http.StatusUnauthorized},
			{f: ReplyBadRequest, statusCode: http.StatusBadRequest},
			{f: ReplyAccessDeniedForbidden, statusCode: http.StatusForbidden, expectedMessage: "access denied: test message"},
			{f: ReplyAccessDeniedUnauthorized, statusCode: http.StatusUnauthorized, expectedMessage: "access denied: test message"},
		}

		for _, c := range cases {
			name := runtime.FuncForPC(reflect.ValueOf(c.f).Pointer()).Name()
			name = name[strings.LastIndex(name, ".")+1:]
			t.Run(name, func(t *testing.T) {
				requestSender := NewIRequestSender(MockTime, SendTimeout(GetTestBusTimeout()), func(requestCtx context.Context, request ibus.Request, responder IResponder) {
					c.f(responder, "test message")
				})
				expectedMessage := "test message"
				if len(c.expectedMessage) > 0 {
					expectedMessage = c.expectedMessage
				}
				meta, resp, err := GetCommandResponse(context.Background(), requestSender, ibus.Request{})
				require.NoError(err)
				require.Equal(ApplicationJSON, meta.ContentType)
				require.Equal(c.statusCode, resp.SysError.HTTPStatus)
				require.Equal(expectedMessage, resp.SysError.Message)
			})
		}

	})
}

func TestHTTP(t *testing.T) {
	require := require.New(t)

	listener, err := net.Listen("tcp", ServerAddress(0))
	require.NoError(err)
	var handler func(w http.ResponseWriter, r *http.Request)
	server := &http.Server{
		Addr: ":0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler(w, r)
		}),
	}
	done := make(chan interface{})
	go func() {
		defer close(done)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			require.NoError(err)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	federationURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	require.NoError(err)
	httpClient, cleanup := NewIHTTPClient()
	defer cleanup()

	t.Run("basic", func(t *testing.T) {
		handler = func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(err)
			w.Write([]byte("hello, " + string(body)))
		}
		resp, err := httpClient.Req(federationURL.String()+"/test", "world")
		require.NoError(err)
		require.Equal("hello, world", resp.Body)
		require.Equal(http.StatusOK, resp.HTTPResp.StatusCode)
	})

	t.Run("cookies & headers", func(t *testing.T) {
		handler = func(_ http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			require.NoError(err)
			// require.Len(r.Header, 2)
			require.Equal("headerValue", r.Header["Header-Key"][0])
			require.Equal("Bearer authorizationValue", r.Header["Authorization"][0])
		}
		resp, err := httpClient.Req(federationURL.String()+"/test", "world",
			WithCookies("cookieKey", "cookieValue"),
			WithHeaders("Header-Key", "headerValue"),
			WithAuthorizeBy("authorizationValue"),
		)
		require.NoError(err)
		fmt.Println(resp.Body)
	})

	require.NoError(server.Shutdown(context.Background()))

	<-done
}
