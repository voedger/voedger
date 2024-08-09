/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

type commandContextStorage struct {
	argFunc         ArgFunc
	unloggedArgFunc UnloggedArgFunc
	wsidFunc        WSIDFunc
	wlogOffsetFunc  WLogOffsetFunc
}

type commandContextKeyBuilder struct {
	baseKeyBuilder
}

func (b *commandContextKeyBuilder) Storage() appdef.QName {
	return sys.Storage_CommandContext
}

func (b *commandContextKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*commandContextKeyBuilder)
	return ok
}

func (s *commandContextStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &commandContextKeyBuilder{}
}
func (s *commandContextStorage) Get(_ istructs.IStateKeyBuilder) (istructs.IStateValue, error) {
	return &cmdContextValue{
		arg:         s.argFunc(),
		unloggedArg: s.unloggedArgFunc(),
		wsid:        s.wsidFunc(),
		wlogOffset:  s.wlogOffsetFunc(),
	}, nil
}

type cmdContextValue struct {
	baseStateValue
	arg         istructs.IObject
	unloggedArg istructs.IObject
	wsid        istructs.WSID
	wlogOffset  istructs.Offset
}

func (v *cmdContextValue) AsInt64(name string) int64 {
	switch name {
	case sys.Storage_CommandContext_Field_Workspace:
		return int64(v.wsid)
	case sys.Storage_CommandContext_Field_WLogOffset:
		return int64(v.wlogOffset)
	}
	return v.baseStateValue.AsInt64(name)
}

func (v *cmdContextValue) AsValue(name string) istructs.IStateValue {
	if name == sys.Storage_CommandContext_Field_ArgumentObject {
		return &objectValue{
			object: v.arg,
		}
	}
	if name == sys.Storage_CommandContext_Field_ArgumentUnloggedObject {
		return &objectValue{
			object: v.unloggedArg,
		}
	}
	return v.baseStateValue.AsValue(name)
}
