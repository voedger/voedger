/*
 * Copyright (c) 2020-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func GetPrincipalTokenFromState(st istructs.IState) (token string, err error) {
	kb, err := st.KeyBuilder(RequestSubject, appdef.NullQName)
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
