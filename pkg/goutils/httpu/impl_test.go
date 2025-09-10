/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package httpu

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTP(t *testing.T) {
	require := require.New(t)

	listener, err := net.Listen("tcp", localhostDynamic)
	require.NoError(err)
	var handler func(w http.ResponseWriter, r *http.Request)
	server := &http.Server{
		Addr: localhostDynamic,
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

	testCases := []struct {
		name      string
		opts      []ReqOptFunc
		setup     func()
		verify    func(t *testing.T, resp *HTTPResponse, req *http.Request)
		customURL string
	}{
		{
			"basic",
			nil,
			func() {
				handler = func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					require.NoError(err)
					_, err = w.Write([]byte("hello, " + string(body)))
					require.NoError(err)
				}
			},
			func(t *testing.T, resp *HTTPResponse, req *http.Request) {
				require.Equal("hello, world", resp.Body)
				require.Equal(http.StatusOK, resp.HTTPResp.StatusCode)
			},
			"",
		},
		{
			"headers & cookies & auth",
			[]ReqOptFunc{
				WithHeaders("Header-Key", "headerValue"),
				WithCookies("cookieKey", "cookieValue"),
				WithAuthorizeBy("authorizationValue"),
			},
			func() {
				handler = func(w http.ResponseWriter, r *http.Request) {
					require.Equal("headerValue", r.Header.Get("Header-Key"))
					require.Equal("Bearer authorizationValue", r.Header.Get("Authorization"))
					cookie, err := r.Cookie("cookieKey")
					require.NoError(err)
					require.Equal("cookieValue", cookie.Value)
					_, err = w.Write([]byte("ok"))
					require.NoError(err)
				}
			},
			func(t *testing.T, resp *HTTPResponse, req *http.Request) {
				require.Equal("ok", resp.Body)
			},
			"",
		},
		{
			"custom method",
			[]ReqOptFunc{WithMethod("PATCH")},
			func() {
				handler = func(w http.ResponseWriter, r *http.Request) {
					require.Equal("PATCH", r.Method)
					_, err := w.Write([]byte("patched"))
					require.NoError(err)
				}
			},
			func(t *testing.T, resp *HTTPResponse, req *http.Request) {
				require.Equal("patched", resp.Body)
			},
			"",
		},
		{
			"error status",
			[]ReqOptFunc{WithExpectedCode(http.StatusTeapot)},
			func() {
				handler = func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusTeapot)
					_, err := w.Write([]byte("i am a teapot"))
					require.NoError(err)
				}
			},
			func(t *testing.T, resp *HTTPResponse, req *http.Request) {
				require.Equal(http.StatusTeapot, resp.HTTPResp.StatusCode)
				require.Equal("i am a teapot", resp.Body)
			},
			"",
		},
		{
			"relative url",
			[]ReqOptFunc{WithRelativeURL("/foo/bar")},
			func() {
				handler = func(w http.ResponseWriter, r *http.Request) {
					require.Equal("/foo/bar", r.URL.Path)
					_, err := w.Write([]byte("relurl"))
					require.NoError(err)
				}
			},
			func(t *testing.T, resp *HTTPResponse, req *http.Request) {
				require.Equal("relurl", resp.Body)
			},
			"/orig",
		},
		{
			"discard response",
			[]ReqOptFunc{WithDiscardResponse()},
			func() {
				handler = func(w http.ResponseWriter, _ *http.Request) {
					_, err := w.Write([]byte("should be discarded"))
					require.NoError(err)
				}
			},
			func(t *testing.T, resp *HTTPResponse, req *http.Request) {
				require.Nil(resp)
			},
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			url := federationURL.String() + "/test"
			if tc.customURL != "" {
				url = federationURL.String() + tc.customURL
			}
			resp, _ := httpClient.Req(context.Background(), url, "world", tc.opts...)
			req := &http.Request{}
			tc.verify(t, resp, req)
		})
	}

	require.NoError(server.Shutdown(context.Background()))
	<-done
}

func TestHTTPReqWithOptions(t *testing.T) {
	require := require.New(t)

	var handler func(w http.ResponseWriter, r *http.Request)

	ts := http.Server{
		Addr: localhostDynamic,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler(w, r)
		}),
	}
	ln, err := net.Listen("tcp", localhostDynamic)
	require.NoError(err)
	go ts.Serve(ln) // nolint errcheck
	defer func() { require.NoError(ts.Shutdown(context.Background())) }()
	url := "http://" + ln.Addr().String()

	httpClient, cleanup := NewIHTTPClient()
	defer cleanup()

	t.Run("multiple headers and cookies", func(t *testing.T) {
		var gotReq *http.Request
		handler = func(w http.ResponseWriter, r *http.Request) {
			gotReq = r
			w.WriteHeader(http.StatusOK)
		}
		_, _ = httpClient.Req(context.Background(), url, "body",
			WithHeaders("A", "a", "B", "b"),
			WithCookies("c1", "v1", "c2", "v2"),
		)
		require.NotNil(gotReq)
		require.Equal("a", gotReq.Header.Get("A"))
		require.Equal("b", gotReq.Header.Get("B"))
		cookieMap := map[string]string{}
		for _, c := range gotReq.Cookies() {
			cookieMap[c.Name] = c.Value
		}
		require.Equal("v1", cookieMap["c1"])
		require.Equal("v2", cookieMap["c2"])
	})

	t.Run("WithoutAuth removes Authorization from headers and cookies", func(t *testing.T) {
		var gotReq *http.Request
		handler = func(w http.ResponseWriter, r *http.Request) {
			gotReq = r
			w.WriteHeader(http.StatusOK)
		}
		_, _ = httpClient.Req(context.Background(), url, "body",
			WithAuthorizeBy("tok"),
			WithCookies(authorization, "tok"),
			WithoutAuth(),
		)
		require.NotNil(gotReq)
		require.Empty(gotReq.Header.Get(authorization))
		for _, c := range gotReq.Cookies() {
			require.NotEqual(authorization, c.Name)
		}
	})

	t.Run("WithResponseHandler is called", func(t *testing.T) {
		var handlerCalled bool
		handler = func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte("ok"))
			require.NoError(err)
		}
		_, _ = httpClient.Req(context.Background(), url, "body", WithResponseHandler(func(resp *http.Response) { handlerCalled = true }))
		require.True(handlerCalled)
	})

	t.Run("WithBodyReader sends custom body", func(t *testing.T) {
		var gotBody string
		handler = func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			w.WriteHeader(http.StatusOK)
		}
		body := strings.NewReader("custom body")
		_, _ = httpClient.ReqReader(context.Background(), url, body)
		require.Equal("custom body", gotBody)
	})

	t.Run("WithLongPolling calls handler on unexpected status", func(t *testing.T) {
		var handlerCalled bool
		handler = func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
			_, err := w.Write([]byte("unexpected"))
			require.NoError(err)
			handlerCalled = true
		}
		_, _ = httpClient.Req(context.Background(), url, "body", WithLongPolling())
		require.True(handlerCalled)
	})

	t.Run("WithMaxRetryDurationOn503 and default WithSkipRetryOn503", func(t *testing.T) {
		var callCount int
		handler = func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_, err := httpClient.Req(context.Background(), url, "body", WithMaxRetryDurationOn503(50*time.Millisecond))
		require.Error(err)
		require.GreaterOrEqual(callCount, 1)
	})

	t.Run("concurrent requests", func(t *testing.T) {
		var count int32
		handler = func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt32(&count, 1)
			_, err := w.Write([]byte("ok"))
			require.NoError(err)
		}
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := httpClient.Req(context.Background(), url, "body")
				require.NoError(err)
				require.Equal("ok", resp.Body)
			}()
		}
		wg.Wait()
		require.Equal(int32(10), count)
	})

	t.Run("default options validation", func(t *testing.T) {
		t.Run("WithDiscardResponse and WithResponseHandler", func(t *testing.T) {
			require.Panics(func() {
				//nolint errcheck
				httpClient.Req(context.Background(), url, "body", WithDiscardResponse(), WithResponseHandler(func(*http.Response) {}))
			})
		})
		t.Run("WithMaxRetryDurationOn503 and WithSkipRetryOn503", func(t *testing.T) {
			require.Panics(func() {
				//nolint errcheck
				httpClient.Req(context.Background(), url, "body", WithMaxRetryDurationOn503(time.Millisecond), WithSkipRetryOn503())
			})
		})
	})
}

func TestContextCanceled(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithCancel(context.Background())

	count := 0
	ts := http.Server{
		Addr: localhostDynamic,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count++
			w.WriteHeader(http.StatusServiceUnavailable)
			if count < 5 {
				return
			}
			cancel()
		}),
	}
	ln, err := net.Listen("tcp", localhostDynamic)
	require.NoError(err)
	go ts.Serve(ln) // nolint errcheck
	defer func() { require.NoError(ts.Shutdown(context.Background())) }()
	url := "http://" + ln.Addr().String()

	httpClient, cleanup := NewIHTTPClient()
	defer cleanup()

	resp, err := httpClient.Req(ctx, url, "body")
	require.ErrorIs(err, context.Canceled)
	require.Nil(resp)
}
