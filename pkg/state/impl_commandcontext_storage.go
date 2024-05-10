/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type commandContextStorage struct {
	argFunc         ArgFunc
	unloggedArgFunc UnloggedArgFunc
	wsidFunc        WSIDFunc
	wlogOffsetFunc  WLogOffsetFunc
}

func (s *commandContextStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(CommandContext, appdef.NullQName)
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
	istructs.IStateValue
	arg         istructs.IObject
	unloggedArg istructs.IObject
	wsid        istructs.WSID
	wlogOffset  istructs.Offset
}

func (v *cmdContextValue) AsInt64(name string) int64 {
	switch name {
	case Field_Workspace:
		return int64(v.wsid)
	case Field_WLogOffset:
		return int64(v.wlogOffset)
	}
	panic(errUndefined(name))
}

func (v *cmdContextValue) AsValue(name string) istructs.IStateValue {
	if name == Field_ArgumentObject {
		return &objectValue{
			object: v.arg,
		}
	}
	if name == Field_ArgumentUnloggedObject {
		return &objectValue{
			object: v.unloggedArg,
		}
	}
	panic(errUndefined(name))
}
