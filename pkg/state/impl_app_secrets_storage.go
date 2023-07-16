/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
)

type appSecretsStorage struct {
	secretReader isecrets.ISecretReader
}

func (s *appSecretsStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(AppSecretsStorage, appdef.NullQName)
}
func (s *appSecretsStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	k := key.(*keyBuilder)
	if _, ok := k.data[Field_Secret]; !ok {
		return nil, fmt.Errorf("'%s': %w", Field_Secret, ErrNotFound)
	}
	bb, e := s.secretReader.ReadSecret(k.data[Field_Secret].(string))
	if errors.Is(e, fs.ErrNotExist) || e == isecrets.ErrSecretNameIsBlank {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &appSecretsStorageValue{
		content:    string(bb),
		toJSONFunc: s.toJSON,
	}, nil
}
func (s *appSecretsStorage) toJSON(sv istructs.IStateValue, _ ...interface{}) (string, error) {
	return fmt.Sprintf(`{"Body":"%s"}`, sv.(*appSecretsStorageValue).content), nil
}
