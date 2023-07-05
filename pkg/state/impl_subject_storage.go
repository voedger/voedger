/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"encoding/json"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
)

type subjectStorage struct {
	principalsFunc PrincipalsFunc
	tokenFunc      TokenFunc
}

func (s *subjectStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(SubjectStorage, appdef.NullQName)
}
func (s *subjectStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	ssv := &subjectStorageValue{
		token:      s.tokenFunc(),
		toJSONFunc: s.toJSON,
	}
	for _, principal := range s.principalsFunc() {
		if principal.Kind == iauthnz.PrincipalKind_Device || principal.Kind == iauthnz.PrincipalKind_User {
			ssv.profileWSID = int64(principal.WSID)
			ssv.name = principal.Name
			if principal.Kind == iauthnz.PrincipalKind_Device {
				ssv.kind = int32(istructs.SubjectKind_Device)
			} else {
				ssv.kind = int32(istructs.SubjectKind_User)
			}
			break
		}
	}
	return ssv, nil
}
func (s *subjectStorage) GetBatch(items []GetBatchItem) (err error) {
	for i := range items {
		items[i].value, err = s.Get(items[i].key)
		if err != nil {
			break
		}
	}
	return
}
func (s *subjectStorage) toJSON(sv istructs.IStateValue, _ ...interface{}) (string, error) {
	value := sv.(*subjectStorageValue)
	obj := make(map[string]interface{})
	obj[Field_ProfileWSID] = value.profileWSID
	obj[Field_Kind] = value.kind
	obj[Field_Name] = value.name
	obj[Field_Token] = value.token
	bb, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(bb), nil
}
