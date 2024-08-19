/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"maps"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/uniques"
)

type uniquesStorage struct {
	uniqiesHandler state.UniquesHandler
	appStructsFunc state.AppStructsFunc
	wsidFunc       state.WSIDFunc
}

type uniquesValue struct {
	baseStateValue
	id istructs.RecordID
}

func (v *uniquesValue) AsInt64(name string) int64 {
	if name == sys.Storage_Uniq_Field_ID {
		return int64(v.id)
	}
	return v.baseStateValue.AsInt64(name)
}

func NewUniquesStorage(appStructsFunc state.AppStructsFunc, wsidFunc state.WSIDFunc, customHandler state.UniquesHandler) state.IStateStorage {
	return &uniquesStorage{
		appStructsFunc: appStructsFunc,
		wsidFunc:       wsidFunc,
		uniqiesHandler: customHandler,
	}
}

func (s *uniquesStorage) NewKeyBuilder(entity appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newUniqKeyBuilder(sys.Storage_Uniq, entity)
}

func (s *uniquesStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	k := key.(*uniqKeyBuilder)
	var id istructs.RecordID
	if s.uniqiesHandler != nil {
		id, err = s.uniqiesHandler(k.entity, s.wsidFunc(), k.data)
	} else {
		id, err = uniques.GetRecordIDByUniqueCombination(s.wsidFunc(), k.entity, s.appStructsFunc(), k.data)
	}
	if err != nil {
		return nil, err
	}
	if id == istructs.NullRecordID {
		return nil, nil
	}
	return &uniquesValue{id: id}, nil
}

type uniqKeyBuilder struct {
	data    map[string]interface{}
	storage appdef.QName
	entity  appdef.QName
}

func newUniqKeyBuilder(storage, entity appdef.QName) *uniqKeyBuilder {
	return &uniqKeyBuilder{
		data:    make(map[string]interface{}),
		storage: storage,
		entity:  entity,
	}
}

func (b *uniqKeyBuilder) Storage() appdef.QName                            { return b.storage }
func (b *uniqKeyBuilder) Entity() appdef.QName                             { return b.entity }
func (b *uniqKeyBuilder) PutInt32(name string, value int32)                { b.data[name] = value }
func (b *uniqKeyBuilder) PutInt64(name string, value int64)                { b.data[name] = value }
func (b *uniqKeyBuilder) PutFloat32(name string, value float32)            { b.data[name] = value }
func (b *uniqKeyBuilder) PutFloat64(name string, value float64)            { b.data[name] = value }
func (b *uniqKeyBuilder) PutBytes(name string, value []byte)               { b.data[name] = value }
func (b *uniqKeyBuilder) PutString(name string, value string)              { b.data[name] = value }
func (b *uniqKeyBuilder) PutQName(name string, value appdef.QName)         { b.data[name] = value }
func (b *uniqKeyBuilder) PutBool(name string, value bool)                  { b.data[name] = value }
func (b *uniqKeyBuilder) PutRecordID(name string, value istructs.RecordID) { b.data[name] = value }
func (b *uniqKeyBuilder) PutNumber(string, float64)                        { panic(ErrNotSupported) }
func (b *uniqKeyBuilder) PutChars(string, string)                          { panic(ErrNotSupported) }
func (b *uniqKeyBuilder) PutFromJSON(j map[string]any)                     { maps.Copy(b.data, j) }
func (b *uniqKeyBuilder) PartitionKey() istructs.IRowWriter                { panic(ErrNotSupported) }
func (b *uniqKeyBuilder) ClusteringColumns() istructs.IRowWriter           { panic(ErrNotSupported) }
func (b *uniqKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	kb, ok := src.(*uniqKeyBuilder)
	if !ok {
		return false
	}
	if b.storage != kb.storage {
		return false
	}
	if b.entity != kb.entity {
		return false
	}
	if !maps.Equal(b.data, kb.data) {
		return false
	}
	return true
}
func (b *uniqKeyBuilder) ToBytes(istructs.WSID) ([]byte, []byte, error) { panic(ErrNotSupported) }
