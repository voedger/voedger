/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func GetPrincipalTokenFromState(st istructs.IState) (token string, err error) {
	kb, err := st.KeyBuilder(sys.Storage_RequestSubject, appdef.NullQName)
	if err != nil {
		return "", err
	}
	principalTokenValue, err := st.MustExist(kb)
	if err != nil {
		return "", err
	}
	token = principalTokenValue.AsString(sys.Storage_RequestSubject_Field_Token)
	return token, nil
}
