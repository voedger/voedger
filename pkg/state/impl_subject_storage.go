/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"encoding/json"

	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

type subjectStorage struct {
	principalsFunc PrincipalsFunc
	tokenFunc      TokenFunc
}

func (s *subjectStorage) NewKeyBuilder(_ schemas.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(SubjectStorage, schemas.NullQName)
}
func (s *subjectStorage) GetBatch(items []GetBatchItem) (err error) {
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
	for i := range items {
		items[i].value = ssv
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
