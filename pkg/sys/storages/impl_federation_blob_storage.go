/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

const readBufferSize = 1024

type federationBlobStorage struct {
	appStructs state.AppStructsFunc
	wsid       state.WSIDFunc
	federation federation.IFederation
	tokens     itokens.ITokens
	emulation  state.FederationBlobHandler
}

func NewFederationBlobStorage(appStructs state.AppStructsFunc, wsid state.WSIDFunc, federation federation.IFederation, tokens itokens.ITokens, emulation state.FederationBlobHandler) state.IStateStorage {
	return &federationBlobStorage{
		appStructs: appStructs,
		wsid:       wsid,
		federation: federation,
		tokens:     tokens,
		emulation:  emulation,
	}
}

type federationBlobKeyBuilder struct {
	baseKeyBuilder
	expectedCodes string
	blobID        istructs.RecordID
	owner         string
	appname       string
	wsid          istructs.WSID
	token         string
}

func (b *federationBlobKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*federationBlobKeyBuilder)
	if !ok {
		return false
	}
	kb := src.(*federationBlobKeyBuilder)
	if b.blobID != kb.blobID {
		return false
	}
	if b.expectedCodes != kb.expectedCodes {
		return false
	}
	if b.owner != kb.owner {
		return false
	}
	if b.appname != kb.appname {
		return false
	}
	if b.wsid != kb.wsid {
		return false
	}
	return true
}

func (b *federationBlobKeyBuilder) PutString(name string, value string) {
	if name == sys.Storage_FederationBlob_Field_ExpectedCodes {
		b.expectedCodes = value
		return
	}
	if name == sys.Storage_FederationBlob_Field_Owner {
		b.owner = value
		return
	}
	if name == sys.Storage_FederationBlob_Field_AppName {
		b.appname = value
		return
	}
	if name == sys.Storage_FederationBlob_Field_Token {
		b.token = value
		return
	}
	b.baseKeyBuilder.PutString(name, value)
}

func (b *federationBlobKeyBuilder) PutInt64(name string, value int64) {
	if name == sys.Storage_FederationBlob_Field_BlobID {
		recordID, err := coreutils.Int64ToRecordID(value)
		if err != nil {
			panic(err)
		}
		b.blobID = recordID
		return
	}
	if name == sys.Storage_FederationBlob_Field_WSID {
		wsid, err := coreutils.Int64ToWSID(value)
		if err != nil {
			panic(err)
		}
		b.wsid = wsid
		return
	}
	b.baseKeyBuilder.PutInt64(name, value)
}

func (s *federationBlobStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &federationBlobKeyBuilder{
		baseKeyBuilder: baseKeyBuilder{storage: sys.Storage_FederationBlob},
	}
}
func (s *federationBlobStorage) getReadCloser(key istructs.IStateKeyBuilder) (io.ReadCloser, error) {
	appqname := s.appStructs().AppQName()

	opts := make([]coreutils.ReqOptFunc, 0)

	kb := key.(*federationBlobKeyBuilder)

	for _, ec := range strings.Split(kb.expectedCodes, ",") {
		if ec == "" {
			continue
		}
		code, err := strconv.Atoi(ec)
		if err != nil {
			return nil, err
		}
		opts = append(opts, coreutils.WithExpectedCode(code))
	}

	if kb.blobID == 0 {
		return nil, errBlobIDNotSpecified
	}

	var owner string

	if kb.owner != "" {
		owner = kb.owner
	} else {
		owner = appqname.Owner()
	}

	var appname string

	if kb.appname != "" {
		appname = kb.appname
	} else {
		appname = appqname.Name()
	}

	var wsid istructs.WSID

	if kb.wsid != 0 {
		wsid = kb.wsid
	} else {
		wsid = s.wsid()
	}

	var readCloser io.ReadCloser

	if s.emulation != nil {
		result, err := s.emulation(owner, appname, wsid, kb.blobID)
		if err != nil {
			return nil, err
		}
		readCloser = io.NopCloser(bytes.NewReader(result))
	} else {
		if kb.token != "" {
			opts = append(opts, coreutils.WithAuthorizeBy(kb.token))
		} else {
			appQName := appdef.NewAppQName(owner, appname)
			systemPrincipalToken, err := payloads.GetSystemPrincipalToken(s.tokens, appQName)
			if err != nil {
				return nil, err
			}
			opts = append(opts, coreutils.WithAuthorizeBy(systemPrincipalToken))
		}
		// FIXME
		blobReader, err := s.federation.ReadBLOB(appdef.NewAppQName(owner, appname), wsid, appdef.NullQName, "", istructs.NullRecordID, opts...)
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
	if name == sys.Storage_FederationBlob_Field_Body {
		return v.data
	}
	return v.baseStateValue.AsBytes(name)
}
