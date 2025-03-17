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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
)

func TestNewHTTPError(t *testing.T) {
	require := require.New(t)
	t.Run("simple", func(t *testing.T) {
		sysErr := NewHTTPError(http.StatusInternalServerError, errors.New("test error"))
		require.Empty(sysErr.Data)
		require.Equal(http.StatusInternalServerError, sysErr.HTTPStatus)
		require.Equal("test error", sysErr.Message)
		require.Equal(appdef.NullQName, sysErr.QName)
		require.Equal(`{"sys.Error":{"HTTPStatus":500,"Message":"test error"}}`, sysErr.ToJSON_APIV1())
	})

	t.Run("formatted", func(t *testing.T) {
		sysErr := NewHTTPErrorf(http.StatusInternalServerError, "test ", "error")
		require.Empty(sysErr.Data)
		require.Equal(http.StatusInternalServerError, sysErr.HTTPStatus)
		require.Equal("test error", sysErr.Message)
		require.Equal(appdef.NullQName, sysErr.QName)
		require.Equal(`{"sys.Error":{"HTTPStatus":500,"Message":"test error"}}`, sysErr.ToJSON_APIV1())
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
