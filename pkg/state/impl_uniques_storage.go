/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys/uniques"
)

type uniquesStorage struct {
	appStructsFunc AppStructsFunc
	wsidFunc       WSIDFunc
}

type uniquesValue struct {
	baseStateValue
	id istructs.RecordID
}

func (v *uniquesValue) AsInt64(name string) int64 {
	if name == Field_ID {
		return int64(v.id)
	}
	panic(errUndefined(name))
}

func newUniquesStorage(appStructsFunc AppStructsFunc, wsidFunc WSIDFunc) *uniquesStorage {
	return &uniquesStorage{
		appStructsFunc: appStructsFunc,
		wsidFunc:       wsidFunc,
	}
}

func (s *uniquesStorage) NewKeyBuilder(entity appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newKeyBuilder(Uniques, entity)
}

func (s *uniquesStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	k := key.(*keyBuilder)
	id, err := uniques.GetRecordIDByUniqueCombination(s.wsidFunc(), k.entity, s.appStructsFunc(), k.data)
	if err != nil {
		return nil, err
	}
	if id == istructs.NullRecordID {
		return nil, nil
	}
	return &uniquesValue{id: id}, nil
}
