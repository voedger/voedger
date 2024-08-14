/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func TestHttpStorage_BasicUsage(t *testing.T) {
	require := require.New(t)
	s := ProvideAsyncActualizerStateFactory()(context.Background(), nilAppStructsFunc, nil, nil, nil, nil, nil, nil, nil, 0, 0)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(http.MethodPost, r.Method)
		require.Equal("my-value", r.Header.Get("my-header"))
		require.Equal("application/json", r.Header.Get("Content-Type"))
		bb, err := io.ReadAll(r.Body)
		require.NoError(err)
		require.Equal(`{"hello":"api"}`, string(bb))
		_, _ = w.Write([]byte(`{"hello":"storage"}`))
	}))
	defer ts.Close()

	k, err := s.KeyBuilder(sys.Storage_Http, appdef.NullQName)
	require.NoError(err)

	k.PutString(sys.Storage_Http_Field_Url, ts.URL)
	k.PutString(sys.Storage_Http_Field_Method, http.MethodPost)
	k.PutString(sys.Storage_Http_Field_Header, "my-header: my-value")
	k.PutString(sys.Storage_Http_Field_Header, "Content-type: application/json")
	k.PutBytes(sys.Storage_Http_Field_Body, []byte(`{"hello":"api"}`))
	require.Equal(fmt.Sprintf(`%s %s {"hello":"api"}`, http.MethodPost, ts.URL), k.(fmt.Stringer).String())
	var v istructs.IStateValue
	err = s.Read(k, func(_ istructs.IKey, value istructs.IStateValue) (err error) {
		v = value
		return
	})
	require.NoError(err)
	require.Equal([]byte(`{"hello":"storage"}`), v.AsBytes(sys.Storage_Http_Field_Body))
	require.Equal(`{"hello":"storage"}`, v.AsString(sys.Storage_Http_Field_Body))
	require.Equal(int32(http.StatusOK), v.AsInt32(sys.Storage_Http_Field_StatusCode))
	require.Contains(v.AsString(sys.Storage_Http_Field_Header), "Content-Length: 19")
	require.Contains(v.AsString(sys.Storage_Http_Field_Header), "Content-Type: text/plain")
}
func TestHttpStorage_Timeout(t *testing.T) {
	t.Run("Should panic when url not found", func(t *testing.T) {
		require := require.New(t)
		s := ProvideAsyncActualizerStateFactory()(context.Background(), nilAppStructsFunc, nil, nil, nil, nil, nil, nil, nil, 0, 0)
		k, err := s.KeyBuilder(sys.Storage_Http, appdef.NullQName)
		require.NoError(err)

		err = s.Read(k, func(istructs.IKey, istructs.IStateValue) error { return nil })
		require.ErrorIs(err, ErrNotFound)
	})
	t.Run("Should return error on timeout", func(t *testing.T) {
		require := require.New(t)
		s := ProvideAsyncActualizerStateFactory()(context.Background(), nilAppStructsFunc, nil, nil, nil, nil, nil, nil, nil, 0, 0)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Millisecond * 200)
		}))
		defer ts.Close()

		k, err := s.KeyBuilder(sys.Storage_Http, appdef.NullQName)
		require.NoError(err)
		k.PutString(sys.Storage_Http_Field_Url, ts.URL)
		k.PutInt64(sys.Storage_Http_Field_HTTPClientTimeoutMilliseconds, 100)

		err = s.Read(k, func(istructs.IKey, istructs.IStateValue) error { return nil })

		require.Error(err)
	})
}
func TestHttpStorage_NewKeyBuilder_should_refresh_key_builder(t *testing.T) {
	require := require.New(t)
	s := &httpStorage{}
	k := s.NewKeyBuilder(appdef.NullQName, nil)
	k.PutString(sys.Storage_Http_Field_Url, "url")
	k.PutString(sys.Storage_Http_Field_Method, http.MethodPost)
	k.PutString(sys.Storage_Http_Field_Header, "my-header: my-value")
	k.PutBytes(sys.Storage_Http_Field_Body, []byte(`{"hello":"api"}`))

	hskb := s.NewKeyBuilder(appdef.NullQName, k).(*httpStorageKeyBuilder)

	require.Equal(sys.Storage_Http, hskb.Storage())
	require.Equal("", hskb.url)
	require.Equal("", hskb.method)
	require.Empty(hskb.headers)
	require.Empty(hskb.body)
}
