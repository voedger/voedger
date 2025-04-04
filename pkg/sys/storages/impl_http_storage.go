/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

var requestNumber int64

type httpStorage struct {
	customClient state.IHTTPClient
}

func NewHTTPStorage(customClient state.IHTTPClient) state.IStateStorage {
	return &httpStorage{
		customClient: customClient,
	}
}

type httpStorageKeyBuilder struct {
	baseKeyBuilder
	timeout      time.Duration
	method       string
	url          string
	body         []byte
	headers      map[string]string
	handleErrors bool
}

func (b *httpStorageKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*httpStorageKeyBuilder)
	if !ok {
		return false
	}
	kb := src.(*httpStorageKeyBuilder)
	if b.timeout != kb.timeout {
		return false
	}
	if b.method != kb.method {
		return false
	}
	if b.handleErrors != kb.handleErrors {
		return false
	}
	if b.url != kb.url {
		return false
	}
	if !bytes.Equal(b.body, kb.body) {
		return false
	}
	if len(b.headers) != len(kb.headers) {
		return false
	}
	for k, v := range b.headers {
		if kv, ok := kb.headers[k]; !ok || kv != v {
			return false
		}
	}
	return true
}

func (b *httpStorageKeyBuilder) PutString(name, value string) {
	switch name {
	case sys.Storage_HTTP_Field_Header:
		trim := func(v string) string { return strings.Trim(v, " \n\r\t") }
		ss := strings.SplitN(value, ":", 2)
		b.headers[trim(ss[0])] = trim(ss[1])
	case sys.Storage_HTTP_Field_Method:
		b.method = value
	case sys.Storage_HTTP_Field_URL:
		b.url = value
	default:
		b.baseKeyBuilder.PutString(name, value)
	}
}

func (b *httpStorageKeyBuilder) String() string {
	ss := make([]string, 0, httpStorageKeyBuilderStringerSliceCap)
	ss = append(ss, b.method)
	ss = append(ss, b.url)
	ss = append(ss, string(b.body))
	return strings.Join(ss, " ")
}

func (b *httpStorageKeyBuilder) PutBool(name string, value bool) {
	switch name {
	case sys.Storage_HTTP_Field_HandleErrors:
		b.handleErrors = value
	default:
		b.baseKeyBuilder.PutBool(name, value)
	}
}

func (b *httpStorageKeyBuilder) PutInt64(name string, value int64) {
	switch name {
	case sys.Storage_HTTP_Field_HTTPClientTimeoutMilliseconds:
		b.timeout = time.Duration(value) * time.Millisecond
	default:
		b.baseKeyBuilder.PutInt64(name, value)
	}
}

func (b *httpStorageKeyBuilder) PutBytes(name string, value []byte) {
	switch name {
	case sys.Storage_HTTP_Field_Body:
		b.body = value
	default:
		b.baseKeyBuilder.PutBytes(name, value)
	}
}

func (s *httpStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &httpStorageKeyBuilder{
		baseKeyBuilder: baseKeyBuilder{storage: sys.Storage_HTTP},
		headers:        make(map[string]string),
	}
}
func (s *httpStorage) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	kb := key.(*httpStorageKeyBuilder)

	if kb.url == "" {
		return fmt.Errorf("'url': %w", ErrNotFound)
	}
	method := http.MethodGet
	if kb.method != "" {
		method = kb.method
	}
	timeout := defaultHTTPClientTimeout
	if kb.timeout != 0 {
		timeout = kb.timeout
	}
	var body io.Reader = nil
	if len(kb.body) > 0 {
		body = bytes.NewReader(kb.body)
	}

	errorResult := func(err error) error {
		if !kb.handleErrors {
			return err
		}
		return callback(nil, &httpValue{
			error: err.Error(),
			body:  []byte(err.Error()),
		})
	}

	if s.customClient != nil {
		resStatus, resBody, resHeaders, err := s.customClient.Request(timeout, method, kb.url, body, kb.headers)
		if err != nil {
			return errorResult(err)
		}
		return callback(nil, &httpValue{
			body:       resBody,
			header:     resHeaders,
			statusCode: resStatus,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, kb.url, body)
	if err != nil {
		return errorResult(err)
	}

	for k, v := range kb.headers {
		req.Header.Add(k, v)
	}
	var reqNumber int64
	if logger.IsVerbose() {
		reqNumber = atomic.AddInt64(&requestNumber, 1)
		logger.Verbose("req ", reqNumber, ": ", method, " ", kb.url, " body: ", string(kb.body))
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errorResult(err)
	}
	defer res.Body.Close()

	bb, err := io.ReadAll(res.Body)
	if err != nil {
		return errorResult(err)
	}

	if logger.IsVerbose() {
		logger.Verbose("resp ", reqNumber, ": ", res.StatusCode, " body: ", string(bb))
	}

	return callback(nil, &httpValue{
		body:       bb,
		header:     res.Header,
		statusCode: res.StatusCode,
	})
}

type httpValue struct {
	istructs.IStateValue
	body       []byte
	header     map[string][]string
	statusCode int
	error      string
}

func (v *httpValue) AsBytes(string) []byte { return v.body }
func (v *httpValue) AsInt32(string) int32  { return int32(v.statusCode) } // nolint G115
func (v *httpValue) AsString(name string) string {
	if name == sys.Storage_HTTP_Field_Header {
		var res strings.Builder
		for k, v := range v.header {
			if len(v) > 0 {
				if res.Len() > 0 {
					res.WriteString("\n")
				}
				res.WriteString(fmt.Sprintf("%s: %s", k, v[0])) // FIXME: len(v)>2 ?
			}
		}
		return res.String()
	}
	if name == sys.Storage_HTTP_Field_Error {
		return v.error
	}
	return string(v.body)
}
