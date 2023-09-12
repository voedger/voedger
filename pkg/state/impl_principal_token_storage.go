/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author: Alisher Nurmanov
 */
package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type principalTokenStorage struct {
	tokenFunc TokenFunc
}

func (p *principalTokenStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	ssv := &principalTokenStorageValue{
		token: p.tokenFunc(),
	}
	return ssv, nil
}

func (p *principalTokenStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newPrincipalTokenKeyBuilder()
}

func (p *principalTokenStorage) Validate([]ApplyBatchItem) (err error) {
	panic(ErrNotSupported)
}

func (p *principalTokenStorage) ApplyBatch([]ApplyBatchItem) (err error) {
	panic(ErrNotSupported)
}

func (p *principalTokenStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	panic(ErrNotSupported)
}

func GetPrincipalTokenFromState(st istructs.IState) (token string, err error) {
	kb, err := st.KeyBuilder(PrincipalTokenStorage, appdef.NullQName)
	if err != nil {
		return "", err
	}
	principalTokenValue, err := st.MustExist(kb)
	if err != nil {
		return "", err
	}
	token = principalTokenValue.AsString(Field_Token)
	return token, nil
}
