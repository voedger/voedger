/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type queryContextStorage struct {
	argFunc  ArgFunc
	wsidFunc WSIDFunc
}

func (s *queryContextStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(QueryContext, appdef.NullQName)
}
func (s *queryContextStorage) Get(_ istructs.IStateKeyBuilder) (istructs.IStateValue, error) {
	return &qryContextValue{
		arg:  s.argFunc(),
		wsid: s.wsidFunc(),
	}, nil
}

type qryContextValue struct {
	istructs.IStateValue
	arg  istructs.IObject
	wsid istructs.WSID
}

func (v *qryContextValue) AsInt64(name string) int64 {
	if name == Field_Workspace {
		return int64(v.wsid)
	}
	panic(errUndefined(name))
}

func (v *qryContextValue) AsValue(name string) istructs.IStateValue {
	if name == Field_ArgumentObject {
		return &objectValue{
			object: v.arg,
		}
	}
	panic(errUndefined(name))
}
