/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

type httpStorage struct {
	customClient IHttpClient
}

type httpStorageKeyBuilder struct {
	baseKeyBuilder
	timeout time.Duration
	method  string
	url     string
	body    []byte
	headers map[string]string
}

func (b *httpStorageKeyBuilder) Storage() appdef.QName {
	return sys.Storage_Http
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
	case sys.Storage_Http_Field_Header:
		trim := func(v string) string { return strings.Trim(v, " \n\r\t") }
		ss := strings.SplitN(value, ":", 2)
		b.headers[trim(ss[0])] = trim(ss[1])
	case sys.Storage_Http_Field_Method:
		b.method = value
	case sys.Storage_Http_Field_Url:
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

func (b *httpStorageKeyBuilder) PutInt64(name string, value int64) {
	switch name {
	case sys.Storage_Http_Field_HTTPClientTimeoutMilliseconds:
		b.timeout = time.Duration(value) * time.Millisecond
	default:
		b.baseKeyBuilder.PutInt64(name, value)
	}
}

func (b *httpStorageKeyBuilder) PutBytes(name string, value []byte) {
	switch name {
	case sys.Storage_Http_Field_Body:
		b.body = value
	default:
		b.baseKeyBuilder.PutBytes(name, value)
	}
}

type IHttpClient interface {
	Request(timeout time.Duration, method, url string, body io.Reader, headers map[string]string) (statusCode int, resBody []byte, resHeaders map[string][]string, err error)
}

func (s *httpStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &httpStorageKeyBuilder{
		headers: make(map[string]string),
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

	if s.customClient != nil {
		resStatus, resBody, resHeaders, err := s.customClient.Request(timeout, method, kb.url, body, kb.headers)
		if err != nil {
			return err
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
		return err
	}

	for k, v := range kb.headers {
		req.Header.Add(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	bb, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return callback(nil, &httpValue{
		body:       bb,
		header:     res.Header,
		statusCode: res.StatusCode,
	})
}
