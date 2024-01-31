/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"encoding/json"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type viewRecordsStorage struct {
	ctx             context.Context
	viewRecordsFunc viewRecordsFunc
	iWorkspaceFunc  iWorkspaceFunc
	wsidFunc        WSIDFunc
	n10nFunc        N10nFunc
}

func (s *viewRecordsStorage) NewKeyBuilder(entity appdef.QName, _ istructs.IStateKeyBuilder) (newKeyBuilder istructs.IStateKeyBuilder) {
	return &viewKeyBuilder{
		IKeyBuilder: s.viewRecordsFunc().KeyBuilder(entity),
		view:        entity,
		wsid:        s.wsidFunc(),
	}
}
func (s *viewRecordsStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	k := key.(*viewKeyBuilder)
	v, err := s.viewRecordsFunc().Get(k.wsid, k.IKeyBuilder)
	if err != nil {
		if err == istructsmem.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return &viewValue{
		value: v,
	}, nil
}

func (s *viewRecordsStorage) GetBatch(items []GetBatchItem) (err error) {
	wsidToItemIdx := make(map[istructs.WSID][]int)
	batches := make(map[istructs.WSID][]istructs.ViewRecordGetBatchItem)
	for itemIdx, item := range items {
		k := item.key.(*viewKeyBuilder)
		wsidToItemIdx[k.wsid] = append(wsidToItemIdx[k.wsid], itemIdx)
		batches[k.wsid] = append(batches[k.wsid], istructs.ViewRecordGetBatchItem{Key: k.IKeyBuilder})
	}
	for wsid, batch := range batches {
		err = s.viewRecordsFunc().GetBatch(wsid, batch)
		if err != nil {
			return
		}
		for i, batchItem := range batch {
			itemIndex := wsidToItemIdx[wsid][i]
			if !batchItem.Ok {
				continue
			}
			items[itemIndex].value = &viewValue{
				value: batchItem.Value,
			}
		}
	}
	return err
}
func (s *viewRecordsStorage) Read(kb istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	cb := func(k istructs.IKey, v istructs.IValue) (err error) {
		return callback(k, &viewValue{
			value: v,
		})
	}
	vrkb := kb.(*viewKeyBuilder)
	return s.viewRecordsFunc().Read(s.ctx, vrkb.wsid, vrkb.IKeyBuilder, cb)
}
func (s *viewRecordsStorage) Validate([]ApplyBatchItem) (err error) { return err }
func (s *viewRecordsStorage) ApplyBatch(items []ApplyBatchItem) (err error) {
	batches := make(map[istructs.WSID][]istructs.ViewKV)
	nn := make(map[n10n]istructs.Offset)
	for _, item := range items {
		k := item.key.(*viewKeyBuilder)
		v := item.value.(*viewValueBuilder)
		batches[k.wsid] = append(batches[k.wsid], istructs.ViewKV{Key: k.IKeyBuilder, Value: v.IValueBuilder})
		if nn[n10n{wsid: k.wsid, view: k.view}] < v.offset {
			nn[n10n{wsid: k.wsid, view: k.view}] = v.offset
		}
	}
	for wsid, batch := range batches {
		err = s.viewRecordsFunc().PutBatch(wsid, batch)
		if err != nil {
			return err
		}
	}
	for n, newOffset := range nn {
		s.n10nFunc(n.view, n.wsid, newOffset)
	}
	return err
}
func (s *viewRecordsStorage) ProvideValueBuilder(kb istructs.IStateKeyBuilder, _ istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	return &viewValueBuilder{
		IValueBuilder: s.viewRecordsFunc().NewValueBuilder(kb.(*viewKeyBuilder).view),
		offset:        istructs.NullOffset,
		toJSONFunc:    s.toJSON,
		entity:        kb.Entity(),
	}
}
func (s *viewRecordsStorage) ProvideValueBuilderForUpdate(kb istructs.IStateKeyBuilder, existingValue istructs.IStateValue, _ istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	return &viewValueBuilder{
		IValueBuilder: s.viewRecordsFunc().UpdateValueBuilder(kb.(*viewKeyBuilder).view, existingValue.(*viewValue).value),
		offset:        istructs.NullOffset,
		toJSONFunc:    s.toJSON,
		entity:        kb.Entity(),
	}
}
func (s *viewRecordsStorage) toJSON(sv istructs.IStateValue, opts ...interface{}) (string, error) {
	options := &ToJSONOptions{make(map[string]bool)}
	for _, opt := range opts {
		opt.(ToJSONOption)(options)
	}

	// obj := make(map[string]interface{})
	// —— nnv, commented. Я не понимаю, зачем здесь поиск контейнера со значением, если его результат никак не используется.
	//		Если бы QName (или определение) найденного контейнера передавалась бы дальше в FieldsToMap или бы изменила бы QName	sv, то это бы объяснило зачем.

	// s.appDefFunc().Def(sv.AsQName(appdef.SystemField_QName)).
	// 	Containers(func(cont appdef.Container) {
	// 		containerName := cont.Name()
	// 		if containerName == appdef.SystemContainer_ViewValue {
	// 			obj = coreutils.FieldsToMap(sv, s.appDefFunc(), coreutils.Filter(func(name string, kind appdef.DataKind) bool {
	// 				return !options.excludedFields[name]
	// 			}))
	// 		}
	// 	})

	obj := coreutils.FieldsToMap(sv, s.iWorkspaceFunc(), coreutils.Filter(func(n string, _ appdef.DataKind) bool {
		return !options.excludedFields[n]
	}))

	bb, err := json.Marshal(obj)
	return string(bb), err
}
