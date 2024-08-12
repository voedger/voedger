/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

type queryContextStorage struct {
	argFunc  ArgFunc
	wsidFunc WSIDFunc
}

type queryContextKeyBuilder struct {
	baseKeyBuilder
}

func (b *queryContextKeyBuilder) Storage() appdef.QName {
	return sys.Storage_QueryContext
}
func (b *queryContextKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*queryContextKeyBuilder)
	return ok
}

func (s *queryContextStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &queryContextKeyBuilder{}
}
func (s *queryContextStorage) Get(_ istructs.IStateKeyBuilder) (istructs.IStateValue, error) {
	return &qryContextValue{
		arg:  s.argFunc(),
		wsid: s.wsidFunc(),
	}, nil
}

type qryContextValue struct {
	baseStateValue
	arg  istructs.IObject
	wsid istructs.WSID
}

func (v *qryContextValue) AsInt64(name string) int64 {
	if name == sys.Storage_QueryContext_Field_Workspace {
		return int64(v.wsid)
	}
	return v.baseStateValue.AsInt64(name)
}

func (v *qryContextValue) AsValue(name string) istructs.IStateValue {
	if name == sys.Storage_QueryContext_Field_ArgumentObject {
		return &objectValue{
			object: v.arg,
		}
	}
	return v.baseStateValue.AsValue(name)
}
