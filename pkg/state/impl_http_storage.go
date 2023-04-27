/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type httpStorage struct{}

func (s *httpStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newHTTPStorageKeyBuilder()
}
func (s *httpStorage) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	kb := key.(*httpStorageKeyBuilder)

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

	return callback(nil, &httpStorageValue{
		body:       bb,
		header:     res.Header,
		statusCode: res.StatusCode,
		toJSONFunc: s.toJSON,
	})
}
func (s *httpStorage) toJSON(sv istructs.IStateValue, _ ...interface{}) (string, error) {
	value := sv.(*httpStorageValue)

	obj := make(map[string]interface{})
	obj[Field_Body] = string(value.body)
	obj[Field_Header] = value.header
	obj[Field_StatusCode] = value.statusCode

	bb, err := json.Marshal(obj)
	return string(bb), err
}
