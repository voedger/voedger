/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

const (
	ContentType = "Content-Type"
)

type FederationCommandHandler = func(owner, appname string, wsid istructs.WSID, command appdef.QName, body string) (statusCode int, newIDs map[string]int64, result string, err error)

type federationCommandStorage struct {
	appStructs AppStructsFunc
	wsid       WSIDFunc
	federation federation.IFederation
	tokens     itokens.ITokens
	emulation  FederationCommandHandler
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
	opts := make([]coreutils.ReqOptFunc, 0)

	kb := key.(*keyBuilder)

	if v, ok := kb.data[Field_ExpectedCodes]; ok {
		for _, ec := range strings.Split(v.(string), ",") {
			code, err := strconv.Atoi(ec)
			if err != nil {
				return nil, err
			}
			opts = append(opts, coreutils.WithExpectedCode(code))
		}
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

	appOwnerAndName := owner + istructs.AppQNameQualifierChar + appname

	relativeUrl := fmt.Sprintf("api/%s/%d/%s", appOwnerAndName, wsid, command)

	var resStatus int
	var resBody string
	var newIDs map[string]int64
	var err error

	if s.emulation != nil {
		resStatus, newIDs, resBody, err = s.emulation(owner, appname, wsid, command, body)
		if err != nil {
			return nil, err
		}
	} else {

		if v, ok := kb.data[Field_Token]; ok {
			opts = append(opts, coreutils.WithAuthorizeBy(v.(string)))
		} else {
			appQName := istructs.NewAppQName(owner, appname)
			systemPrincipalToken, err := payloads.GetSystemPrincipalToken(s.tokens, appQName)
			if err != nil {
				return nil, err
			}
			opts = append(opts, coreutils.WithAuthorizeBy(systemPrincipalToken))
		}

		resp, err := s.federation.Func(relativeUrl, body, opts...)
		if err != nil {
			return nil, err
		}
		resBody = resp.Body
		newIDs = resp.NewIDs
		resStatus = resp.HTTPResp.StatusCode
	}

	result := map[string]interface{}{}
	err = json.Unmarshal([]byte(resBody), &result)
	if err != nil {
		return nil, err
	}

	return &fcCmdValue{
		statusCode: resStatus,
		newIds:     &fcCmdNewIds{newIds: newIDs},
		result:     &jsonValue{json: result},
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
	if name == Field_Result {
		return v.result
	}
	panic(errUndefined(name))
}

type fcCmdNewIds struct {
	baseStateValue
	newIds map[string]int64
}

func (v *fcCmdNewIds) AsInt64(name string) int64 {
	if id, ok := v.newIds[name]; ok {
		return id
	}
	panic(errUndefined(name))
}
