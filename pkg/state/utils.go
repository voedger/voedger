/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package state

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

func PopulateKeys(kb istructs.IRowWriter, keys map[string]any) {
	for k, v := range keys {
		switch t := v.(type) {
		case int8:
			kb.PutNumber(k, float64(t))
		case int16:
			kb.PutNumber(k, float64(t))
		case int32:
			kb.PutNumber(k, float64(t))
		case int64:
			kb.PutInt64(k, t)
		case int:
			kb.PutNumber(k, float64(t))
		case float32:
			kb.PutFloat32(k, t)
		case float64:
			kb.PutFloat64(k, t)
		case []byte:
			kb.PutBytes(k, t)
		case string:
			kb.PutString(k, t)
		case appdef.QName:
			kb.PutQName(k, t)
		case bool:
			kb.PutBool(k, t)
		case istructs.RecordID:
			kb.PutRecordID(k, t)
		}
	}
}
