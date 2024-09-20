/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func NewAppSecretsStorage(secretReader isecrets.ISecretReader) state.IStateStorage {
	return &appSecretsStorage{
		secretReader: secretReader,
	}
}

type appSecretsStorage struct {
	secretReader isecrets.ISecretReader
}

type appSecretsStorageKeyBuilder struct {
	baseKeyBuilder
	secret string
}

func (b *appSecretsStorageKeyBuilder) String() string {
	return fmt.Sprintf("%s, secret:%s", b.baseKeyBuilder.String(), b.secret)
}
func (b *appSecretsStorageKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	kb, ok := src.(*appSecretsStorageKeyBuilder)
	if !ok {
		return false
	}
	if b.secret != kb.secret {
		return false
	}
	return true
}
func (b *appSecretsStorageKeyBuilder) PutString(name string, value string) {
	if name == sys.Storage_AppSecretField_Secret {
		b.secret = value
		return
	}
	b.baseKeyBuilder.PutString(name, value)
}

type appSecretValue struct {
	baseStateValue
	content string
}

func (v *appSecretValue) AsString(name string) string {
	return v.content
}

func (s *appSecretsStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &appSecretsStorageKeyBuilder{
		baseKeyBuilder: baseKeyBuilder{storage: sys.Storage_AppSecret},
	}
}
func (s *appSecretsStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	k := key.(*appSecretsStorageKeyBuilder)
	if k.secret == "" {
		return nil, errors.New("secret name is not specified")
	}
	bb, e := s.secretReader.ReadSecret(k.secret)
	if errors.Is(e, fs.ErrNotExist) || errors.Is(e, isecrets.ErrSecretNameIsBlank) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &appSecretValue{
		content: string(bb),
	}, nil
}
