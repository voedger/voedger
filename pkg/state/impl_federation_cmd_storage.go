/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

const (
	Authorization   = "Authorization"
	ContentType     = "Content-Type"
	ApplicationJSON = "application/json"
	BearerPrefix    = "Bearer "
	commandTimeout  = 10 * time.Second
)

type federationCommandStorage struct {
	customClient           IHttpClient
	federationUrl          FederationURL
	appStructs             AppStructsFunc
	wsid                   WSIDFunc
	defaultFederationToken TokenFunc
}

func (s *federationCommandStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(FederationCommand, appdef.NullQName)
}
func (s *federationCommandStorage) Get(key istructs.IStateKeyBuilder) (istructs.IStateValue, error) {
	appqname := s.appStructs().AppQName()
	var owner string
	var appname string
	var wsid istructs.WSID
	var command appdef.QName
	var body string
	var headers map[string]string = make(map[string]string)
	kb := key.(*keyBuilder)

	isExpectedCode := func(code int) bool {
		if v, ok := kb.data[Field_ExpectedCodes]; ok {
			expectedCodes := strings.Split(v.(string), ",")
			for _, ec := range expectedCodes {
				if ec == fmt.Sprint(code) {
					return true
				}
			}
			return false
		}
		return code == http.StatusOK
	}

	if v, ok := kb.data[Field_Owner]; ok {
		owner = v.(string)
	} else {
		owner = appqname.Owner()
	}

	if v, ok := kb.data[Field_AppName]; ok {
		appname = v.(string)
	} else {
		appname = appqname.Name()
	}

	if v, ok := kb.data[Field_WSID]; ok {
		wsid = v.(istructs.WSID)
	} else {
		wsid = s.wsid()
	}

	if v, ok := kb.data[Field_Command]; ok {
		command = v.(appdef.QName)
	} else {
		return nil, errCommandNotSpecified
	}

	if v, ok := kb.data[Field_Body]; ok {
		body = v.(string)
	}
	bodyReader := bytes.NewReader([]byte(body))

	appOwnerAndName := owner + istructs.AppQNameQualifierChar + appname

	url := fmt.Sprintf("%s/api/%s/%d/%s", s.federationUrl(), appOwnerAndName, wsid, command)

	headers[Authorization] = BearerPrefix + s.defaultFederationToken()

	var resStatus int
	var resBody []byte
	var err error

	if s.customClient != nil {
		resStatus, resBody, _, err = s.customClient.Request(commandTimeout, http.MethodPost, url, bodyReader, headers)
		if err != nil {
			return nil, err
		}
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
		if err != nil {
			return nil, err
		}

		for k, v := range headers {
			req.Header.Add(k, v)
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		resBody, err = io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		resStatus = res.StatusCode
	}

	if !isExpectedCode(resStatus) {
		return nil, fmt.Errorf("unexpected status code %d", resStatus)
	}
	respData := map[string]interface{}{}
	err = json.Unmarshal(resBody, &respData)
	if err != nil {
		return nil, err
	}

	return &fcCmdValue{
		statusCode: resStatus,
		newIds:     &fcCmdNewIds{newIds: respData["NewIDs"].(map[string]interface{})},
		result:     &jsonValue{json: respData["Result"].(map[string]interface{})},
	}, nil
}
func (s *federationCommandStorage) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	v, err := s.Get(key)
	if err != nil {
		return err
	}
	return callback(nil, v)
}

type fcCmdValue struct {
	baseStateValue
	statusCode int
	newIds     istructs.IStateValue
	result     istructs.IStateValue
}

func (v *fcCmdValue) AsInt32(name string) int32 {
	if name == Field_StatusCode {
		return int32(v.statusCode)
	}
	panic(errUndefined(name))
}

func (v *fcCmdValue) AsValue(name string) istructs.IStateValue {
	if name == Field_NewIDs {
		return v.newIds
	}
	panic(errUndefined(name))
}

type fcCmdNewIds struct {
	baseStateValue
	newIds map[string]interface{}
}

func (v *fcCmdNewIds) AsInt64(name string) int64 {
	if id, ok := v.newIds[name]; ok {
		return int64(id.(float64))
	}
	panic(errUndefined(name))
}
