/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
)

const readBufferSize = 1024

type FederationBlobHandler = func(owner, appname string, wsid istructs.WSID, blobId int64) (result []byte, err error)
type federationBlobStorage struct {
	appStructs AppStructsFunc
	wsid       WSIDFunc
	federation federation.IFederation
	tokens     itokens.ITokens
	emulation  FederationBlobHandler
}

func (s *federationBlobStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(FederationBlob, appdef.NullQName)
}
func (s *federationBlobStorage) getReadCloser(key istructs.IStateKeyBuilder) (io.ReadCloser, error) {
	appqname := s.appStructs().AppQName()
	var owner string
	var appname string
	var wsid istructs.WSID
	var blobId int64
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

	if v, ok := kb.data[Field_BlobID]; ok {
		blobId = v.(int64)
	} else {
		return nil, errBlobIDNotSpecified
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
		wsid = istructs.WSID(v.(int64))
	} else {
		wsid = s.wsid()
	}

	var readCloser io.ReadCloser

	if s.emulation != nil {
		result, err := s.emulation(owner, appname, wsid, blobId)
		if err != nil {
			return nil, err
		}
		readCloser = io.NopCloser(bytes.NewReader(result))
	} else {
		if v, ok := kb.data[Field_Token]; ok {
			opts = append(opts, coreutils.WithAuthorizeBy(v.(string)))
		} else {
			appQName := appdef.NewAppQName(owner, appname)
			systemPrincipalToken, err := payloads.GetSystemPrincipalToken(s.tokens, appQName)
			if err != nil {
				return nil, err
			}
			opts = append(opts, coreutils.WithAuthorizeBy(systemPrincipalToken))
		}
		blobReader, err := s.federation.ReadBLOB(appdef.NewAppQName(owner, appname), wsid, istructs.RecordID(blobId), opts...)
		if err != nil {
			return nil, err
		}
		readCloser = blobReader
	}
	return readCloser, nil
}
func (s *federationBlobStorage) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	readCloser, err := s.getReadCloser(key)
	if err != nil {
		return err
	}

	defer readCloser.Close()
	buffer := make([]byte, readBufferSize)
	var n int
	for err == nil {
		n, err = readCloser.Read(buffer)
		if err != nil && !errors.Is(err, io.EOF) {
			break
		}
		if n == 0 {
			err = nil
			break
		}
		if err = callback(nil, &fBlobValue{data: buffer[:n]}); err != nil {
			break
		}
	}
	return err
}

type fBlobValue struct {
	baseStateValue
	data []byte
}

func (v *fBlobValue) AsBytes(name string) []byte {
	if name == Field_Body {
		return v.data
	}
	panic(errUndefined(name))
}
