/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestFederationFunc(t *testing.T) {
	require := require.New(t)

	listener, err := net.Listen("tcp", httpu.LocalhostDynamic())
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
	federation, cleanup := New(context.Background(), func() *url.URL {
		return federationURL
	}, coreutils.NilAdminPortGetter, httpu.DefaultRetryPolicyOpts)
	defer cleanup()

	t.Run("basic", func(t *testing.T) {
		handler = func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(err)
			require.JSONEq(`{"fld":"val"}`, string(body))
			_, err = w.Write([]byte(`{
				"newIDs":{"1":2},
				"sections":[{"type":"","elements":[[[["hello, world"]]]]}],
				"currentWLogOffset":13,
				"result":{"Int":42,"Str":"Str","sys.Container":"","sys.QName":"app1pkg.TestCmdResult"}
			}`))
			require.NoError(err)
		}
		resp, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`)
		require.NoError(err)
		resp.Println()
		require.Equal(istructs.Offset(13), resp.CurrentWLogOffset)
		require.Equal("hello, world", resp.SectionRow()[0].(string))
		require.Equal(map[string]interface{}{
			"Int":           float64(42),
			"Str":           "Str",
			"sys.Container": "",
			"sys.QName":     "app1pkg.TestCmdResult",
		}, resp.CmdResult)
		require.Equal(istructs.RecordID(2), resp.NewID())
	})

	t.Run("unexpected error", func(t *testing.T) {
		cases := []struct {
			name        string
			handler     func(body string, w http.ResponseWriter, r *http.Request)
			expectedErr error
			opts        []httpu.ReqOptFunc
		}{
			{
				name: "basic error",
				handler: func(body string, w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte(`{"sys.Error":{"HTTPStatus":500,"Message":"something gone wrong","QName":"sys.SomeErrorQName","Data":"additional data"}}`))
					require.NoError(err)
				},
				expectedErr: coreutils.SysError{
					HTTPStatus: 500,
					QName:      appdef.NewQName("sys", "SomeErrorQName"),
					Message:    "something gone wrong",
					Data:       "additional data",
				},
			},
			{
				name: "wrong QName",
				handler: func(body string, w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte(`{"sys.Error":{"HTTPStatus":500,"Message":"something gone wrong","QName":"errored QName","Data":"additional data"}}`))
					require.NoError(err)
				},
				expectedErr: errors.New(`failed to unmarshal response body: convert error: string «errored QName» to qualified name. Body:
{"sys.Error":{"HTTPStatus":500,"Message":"something gone wrong","QName":"errored QName","Data":"additional data"}}`),
			},
			{
				name: "wrong response JSON",
				handler: func(body string, w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err = w.Write([]byte(`wrong JSON`))
					require.NoError(err)
				},
				expectedErr: errors.New("failed to unmarshal response body: invalid character 'w' looking for beginning of value. Body:\nwrong JSON"),
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				handler = func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					require.NoError(err)
					c.handler(string(body), w, r)
				}
				resp, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`, c.opts...)
				var se coreutils.SysError
				if errors.As(err, &se) {
					require.Equal(c.expectedErr, se, c.name)
				} else {
					require.Equal(c.expectedErr.Error(), err.Error(), c.name)
				}
				log.Println(err.Error())
				require.Nil(resp)
			})
		}
	})

	t.Run("expected error", func(t *testing.T) {
		t.Run("basic", func(t *testing.T) {
			handler = func(w http.ResponseWriter, r *http.Request) {
				_, err := io.ReadAll(r.Body)
				require.NoError(err)
				w.WriteHeader(http.StatusInternalServerError)
				_, err = w.Write([]byte(`{"sys.Error":{"HTTPStatus":500,"Message":"something gone wrong","QName":"sys.SomeErrorQName","Data":"additional data"}}`))
				require.NoError(err)
			}
			resp, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`, httpu.WithExpectedCode(http.StatusInternalServerError))
			require.NoError(err)
			resp.Println()
		})
	})

	t.Run("sections", func(t *testing.T) {
		handler = func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			require.NoError(err)
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(`{"sections":[{"type":"","elements":[[[["Hello", "world"]]],[[["next"]]]]}]}`))
			require.NoError(err)
		}
		resp, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`, httpu.WithExpectedCode(http.StatusInternalServerError))
		require.NoError(err)
		resp.Println()
		require.Equal("Hello", resp.SectionRow()[0].(string))
		require.Equal("world", resp.SectionRow()[1].(string))
		require.Equal("next", resp.SectionRow(1)[0].(string))
	})

	t.Run("no automatic retry on 503", func(t *testing.T) {
		handler = func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			require.NoError(err)
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`, httpu.Expect503())
		require.NoError(err)
	})

	t.Run("discard response", func(t *testing.T) {
		handler = func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			require.NoError(err)
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte(`{"sections":[{"type":"","elements":[[[["Hello", "world"]]],[[["next"]]]]}]}`))
			require.NoError(err)
		}
		resp, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`, httpu.WithDiscardResponse())
		require.NoError(err)
		require.Nil(resp)
	})
	t.Run("context cancel during retry on status", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		federation, cleanup := New(ctx, func() *url.URL {
			return federationURL
		}, coreutils.NilAdminPortGetter, httpu.DefaultRetryPolicyOpts)
		defer cleanup()
		counter := 0
		handler = func(w http.ResponseWriter, _ *http.Request) {
			counter++
			w.WriteHeader(http.StatusServiceUnavailable)
			if counter < 5 {
				return
			}
			cancel()
		}
		_, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`, httpu.ReqOptFunc(httpu.WithRetryOnStatus(http.StatusServiceUnavailable)))
		require.ErrorIs(err, context.Canceled)
	})
}

func TestPanicOnGETAndDiscardResponse(t *testing.T) {
	require := require.New(t)
	federationURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", 123))
	require.NoError(err)
	federation, cleanup := New(context.Background(), func() *url.URL {
		return federationURL
	}, coreutils.NilAdminPortGetter, httpu.DefaultRetryPolicyOpts)
	defer cleanup()

	require.Panics(func() {
		//nolint errcheck
		federation.Func("abc", "", httpu.WithMethod(http.MethodGet), httpu.WithDiscardResponse())
	})
}
