/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

type jobContextStorage struct {
	wsidFunc     state.WSIDFunc
	unixTimeFunc state.UnixTimeFunc
}

func NewJobContextStorage(wsidFunc state.WSIDFunc, unixTimeFunc state.UnixTimeFunc) state.IStateStorage {
	return &jobContextStorage{
		unixTimeFunc: unixTimeFunc,
		wsidFunc:     wsidFunc,
	}
}

type jobContextKeyBuilder struct {
	baseKeyBuilder
}

func (b *jobContextKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*jobContextKeyBuilder)
	return ok
}

func (s *jobContextStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &jobContextKeyBuilder{
		baseKeyBuilder: baseKeyBuilder{storage: sys.Storage_JobContext},
	}
}
func (s *jobContextStorage) Get(_ istructs.IStateKeyBuilder) (istructs.IStateValue, error) {
	return &jobContextValue{
		unixTime: s.unixTimeFunc(),
		wsid:     s.wsidFunc(),
	}, nil
}

type jobContextValue struct {
	baseStateValue
	unixTime int64
	wsid     istructs.WSID
}

func (v *jobContextValue) AsInt64(name string) int64 {
	if name == sys.Storage_JobContext_Field_Workspace {
		return int64(v.wsid) // nolint G115
	}
	if name == sys.Storage_JobContext_Field_UnixTime {
		return v.unixTime
	}
	return v.baseStateValue.AsInt64(name)
}
