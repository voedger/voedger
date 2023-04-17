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
	"github.com/voedger/voedger/pkg/istructs"
)

func TestHttpStorage_BasicUsage(t *testing.T) {
	require := require.New(t)
	s := ProvideAsyncActualizerStateFactory()(context.Background(), &nilAppStructs{}, nil, nil, nil, nil, 0, 0)
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

	k, err := s.KeyBuilder(HTTPStorage, istructs.NullQName)
	require.Nil(err)

	k.PutString(Field_Url, ts.URL)
	k.PutString(Field_Method, http.MethodPost)
	k.PutString(Field_Header, "my-header: my-value")
	k.PutString(Field_Header, "Content-type: application/json")
	k.PutBytes(Field_Body, []byte(`{"hello":"api"}`))
	require.Equal(fmt.Sprintf(`%s %s {"hello":"api"}`, http.MethodPost, ts.URL), k.(fmt.Stringer).String())
	var v istructs.IStateValue
	_ = s.Read(k, func(_ istructs.IKey, value istructs.IStateValue) (err error) {
		v = value
		return
	})
	require.Equal([]byte(`{"hello":"storage"}`), v.AsBytes(Field_Body))
	require.Equal(`{"hello":"storage"}`, v.AsString(Field_Body))
	require.Equal(int32(http.StatusOK), v.AsInt32(Field_StatusCode))
	require.Contains(v.AsString(Field_Header), "Content-Length: 19")
	require.Contains(v.AsString(Field_Header), "Content-Type: text/plain")
	json, err := v.ToJSON()
	require.NoError(err)
	require.NotEmpty(json)
}
func TestHttpStorage_Timeout(t *testing.T) {
	t.Run("Should panic when url not found", func(t *testing.T) {
		require := require.New(t)
		s := ProvideAsyncActualizerStateFactory()(context.Background(), &nilAppStructs{}, nil, nil, nil, nil, 0, 0)
		k, err := s.KeyBuilder(HTTPStorage, istructs.NullQName)
		require.NoError(err)

		require.ErrorIs(errorFromPanic(func() { _ = s.Read(k, func(istructs.IKey, istructs.IStateValue) error { return nil }) }), ErrNotFound)
	})
	t.Run("Should return error on timeout", func(t *testing.T) {
		require := require.New(t)
		s := ProvideAsyncActualizerStateFactory()(context.Background(), &nilAppStructs{}, nil, nil, nil, nil, 0, 0)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Millisecond * 200)
		}))
		defer ts.Close()

		k, err := s.KeyBuilder(HTTPStorage, istructs.NullQName)
		require.NoError(err)
		k.PutString(Field_Url, ts.URL)
		k.PutInt64(Field_HTTPClientTimeoutMilliseconds, 100)

		err = s.Read(k, func(istructs.IKey, istructs.IStateValue) error { return nil })

		require.Error(err)
	})
}
func TestHttpStorage_NewKeyBuilder_should_refresh_key_builder(t *testing.T) {
	require := require.New(t)
	s := &httpStorage{}
	k := s.NewKeyBuilder(istructs.NullQName, nil)
	k.PutString(Field_Url, "url")
	k.PutString(Field_Method, http.MethodPost)
	k.PutString(Field_Header, "my-header: my-value")
	k.PutBytes(Field_Body, []byte(`{"hello":"api"}`))

	hskb := s.NewKeyBuilder(istructs.NullQName, k).(*httpStorageKeyBuilder)

	require.Equal(HTTPStorage, hskb.storage)
	require.Empty(hskb.data)
}
