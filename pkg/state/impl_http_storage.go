/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type httpStorage struct {
	customClient IHttpClient
}

type IHttpClient interface {
	Request(timeout time.Duration, method, url string, body io.Reader, headers map[string]string) (statusCode int, resBody []byte, resHeaders map[string][]string, err error)
}

func (s *httpStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newHttpKeyBuilder()
}
func (s *httpStorage) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	kb := key.(*httpKeyBuilder)

	if s.customClient != nil {
		resStatus, resBody, resHeaders, err := s.customClient.Request(kb.timeout(), kb.method(), kb.url(), kb.body(), kb.headers)
		if err != nil {
			return err
		}
		return callback(nil, &httpValue{
			body:       resBody,
			header:     resHeaders,
			statusCode: resStatus,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), kb.timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, kb.method(), kb.url(), kb.body())
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
