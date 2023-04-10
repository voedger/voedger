/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/untillpro/voedger/pkg/isecrets"
	"github.com/untillpro/voedger/pkg/istructs"
)

type appSecretsStorage struct {
	secretReader isecrets.ISecretReader
}

func (s *appSecretsStorage) NewKeyBuilder(istructs.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(AppSecretsStorage, istructs.NullQName)
}
func (s *appSecretsStorage) GetBatch(items []GetBatchItem) (err error) {
	for i, item := range items {
		k := item.key.(*keyBuilder)
		if _, ok := k.data[Field_Secret]; !ok {
			return fmt.Errorf("'%s': %w", Field_Secret, ErrNotFound)
		}
		bb, e := s.secretReader.ReadSecret(k.data[Field_Secret].(string))
		if errors.Is(e, fs.ErrNotExist) || e == isecrets.ErrSecretNameIsBlank {
			continue
		}
		if e != nil {
			return e
		}
		items[i].value = &appSecretsStorageValue{
			content:    string(bb),
			toJSONFunc: s.toJSON,
		}
	}
	return
}
func (s *appSecretsStorage) toJSON(sv istructs.IStateValue, _ ...interface{}) (string, error) {
	return fmt.Sprintf(`{"Body":"%s"}`, sv.(*appSecretsStorageValue).content), nil
}
