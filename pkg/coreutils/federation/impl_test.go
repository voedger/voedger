/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
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
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestFederationFunc(t *testing.T) {
	require := require.New(t)

	listener, err := net.Listen("tcp", coreutils.ServerAddress(0))
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
	federation, cleanup := New(func() *url.URL {
		return federationURL
	}, coreutils.NilAdminPortGetter)
	defer cleanup()

	t.Run("basic", func(t *testing.T) {
		handler = func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(err)
			require.Equal(`{"fld":"val"}`, string(body))
			w.Write([]byte(`{
				"NewIDs":{"1":2},
				"sections":[{"type":"","elements":[[[["hello, world"]]]]}],
				"CurrentWLogOffset":13,
				"Result":{"Int":42,"Str":"Str","sys.Container":"","sys.QName":"app1pkg.TestCmdResult"}
			}`))
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
			opts        []coreutils.ReqOptFunc
		}{
			{
				name: "basic error",
				handler: func(body string, w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"sys.Error":{"HTTPStatus":500,"Message":"something gone wrong","QName":"sys.SomeErrorQName","Data":"additional data"}}`))
				},
				expectedErr: coreutils.FuncError{
					SysError: coreutils.SysError{
						HTTPStatus: 500,
						QName:      appdef.NewQName("sys", "SomeErrorQName"),
						Message:    "something gone wrong",
						Data:       "additional data",
					},
					ExpectedHTTPCodes: []int{http.StatusOK},
				},
			},
			{
				name: "wrong QName",
				handler: func(body string, w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"sys.Error":{"HTTPStatus":500,"Message":"something gone wrong","QName":"errored QName","Data":"additional data"}}`))
				},
				expectedErr: coreutils.FuncError{
					SysError: coreutils.SysError{
						HTTPStatus: 500,
						QName:      appdef.NewQName("<err>", "errored QName"),
						Message:    "something gone wrong",
						Data:       "additional data",
					},
					ExpectedHTTPCodes: []int{http.StatusOK},
				},
			},
			{
				name: "wrong response JSON",
				handler: func(body string, w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`wrong JSON`))
				},
				expectedErr: errors.New("invalid character 'w' looking for beginning of value"),
			},
			{
				name: "non-OK status is expected",
				handler: func(body string, w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"sys.Error":{"HTTPStatus":500,"Message":"something gone wrong","QName":"errored QName","Data":"additional data"}}`))
				},
				expectedErr: coreutils.FuncError{
					SysError: coreutils.SysError{
						HTTPStatus: http.StatusOK,
					},
					ExpectedHTTPCodes: []int{http.StatusInternalServerError},
				},
				opts: []coreutils.ReqOptFunc{coreutils.WithExpectedCode(http.StatusInternalServerError)},
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
				var fe coreutils.FuncError
				if errors.As(err, &fe) {
					require.Equal(c.expectedErr, err)
				} else {
					require.Equal(c.expectedErr.Error(), err.Error())
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
				w.Write([]byte(`{"sys.Error":{"HTTPStatus":500,"Message":"something gone wrong","QName":"sys.SomeErrorQName","Data":"additional data"}}`))
			}
			resp, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`, coreutils.WithExpectedCode(http.StatusInternalServerError))
			require.NoError(err)
			resp.Println()
			resp.RequireContainsError(t, "something")
			resp.RequireError(t, "something gone wrong")
		})
		t.Run("ExpectedErrorContains", func(t *testing.T) {
			errorMessage := "non-expected"
			handler = func(w http.ResponseWriter, r *http.Request) {
				_, err := io.ReadAll(r.Body)
				require.NoError(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf(`{"sys.Error":{"HTTPStatus":500,"Message":"%s","QName":"sys.SomeErrorQName","Data":"additional data"}}`,
					errorMessage)))
			}
			resp, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`, coreutils.WithExpectedCode(http.StatusInternalServerError,
				"expected error message"))
			require.Error(err)
			require.Nil(resp)

			errorMessage = "expected error message"
			resp, err = federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`, coreutils.WithExpectedCode(http.StatusInternalServerError,
				"expected error message"))
			require.NoError(err)
			resp.RequireContainsError(t, "expected")
			resp.RequireError(t, "expected error message")
		})
	})

	t.Run("sections", func(t *testing.T) {
		handler = func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			require.NoError(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"sections":[{"type":"","elements":[[[["Hello", "world"]]],[[["next"]]]]}]}`))
		}
		resp, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`, coreutils.WithExpectedCode(http.StatusInternalServerError))
		require.NoError(err)
		resp.Println()
		require.Equal("Hello", resp.SectionRow()[0].(string))
		require.Equal("world", resp.SectionRow()[1].(string))
		require.Equal("next", resp.SectionRow(1)[0].(string))
	})

	t.Run("automatic retry on 503", func(t *testing.T) {
		statusCode := http.StatusServiceUnavailable
		handler = func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			require.NoError(err)
			w.WriteHeader(statusCode)
			if statusCode == http.StatusOK {
				w.Write([]byte(`{"sections":[{"type":"","elements":[[[["Hello", "world"]]],[[["next"]]]]}]}`))
			}
			statusCode = http.StatusOK
		}
		resp, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`)
		require.NoError(err)
		resp.Println()
	})

	t.Run("discard response", func(t *testing.T) {
		handler = func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			require.NoError(err)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"sections":[{"type":"","elements":[[[["Hello", "world"]]],[[["next"]]]]}]}`))
		}
		resp, err := federation.Func("/api/123456789/c.sys.CUD", `{"fld":"val"}`, coreutils.WithDiscardResponse())
		require.NoError(err)
		require.Nil(resp)
	})
}
