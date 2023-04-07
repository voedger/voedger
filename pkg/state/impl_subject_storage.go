/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"fmt"

	"github.com/untillpro/voedger/pkg/iauthnz"
	"github.com/untillpro/voedger/pkg/istructs"
)

type subjectStorage struct {
	principalsFunc func() []iauthnz.Principal
}

func (s *subjectStorage) NewKeyBuilder(_ istructs.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(SubjectStorage, istructs.NullQName)
}
func (s *subjectStorage) GetBatch(items []GetBatchItem) (err error) {
	ssv := &subjectStorageValue{toJSONFunc: s.toJSON}
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
	return fmt.Sprintf(`{"%s":%d,"%s":%d,"%s":"%s"}`, Field_ProfileWSID, value.profileWSID, Field_Kind, value.kind, Field_Name, value.name), nil
}
