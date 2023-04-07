/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"encoding/json"

	"github.com/untillpro/voedger/pkg/istructs"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

type viewRecordsStorage struct {
	ctx             context.Context
	viewRecordsFunc viewRecordsFunc
	schemasFunc     schemasFunc
	wsidFunc        WSIDFunc
	n10nFunc        N10nFunc
}

func (s *viewRecordsStorage) NewKeyBuilder(entity istructs.QName, _ istructs.IStateKeyBuilder) (newKeyBuilder istructs.IStateKeyBuilder) {
	return &viewRecordsKeyBuilder{
		IKeyBuilder: s.viewRecordsFunc().KeyBuilder(entity),
		view:        entity,
		wsid:        s.wsidFunc(),
	}
}
func (s *viewRecordsStorage) GetBatch(items []GetBatchItem) (err error) {
	wsidToItemIdx := make(map[istructs.WSID][]int)
	batches := make(map[istructs.WSID][]istructs.ViewRecordGetBatchItem)
	for itemIdx, item := range items {
		k := item.key.(*viewRecordsKeyBuilder)
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
			items[itemIndex].value = &viewRecordsStorageValue{
				value:      batchItem.Value,
				toJSONFunc: s.toJSON,
			}
		}
	}
	return err
}
func (s *viewRecordsStorage) Read(kb istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	cb := func(k istructs.IKey, v istructs.IValue) (err error) {
		return callback(k, &viewRecordsStorageValue{
			value:      v,
			toJSONFunc: s.toJSON,
		})
	}
	vrkb := kb.(*viewRecordsKeyBuilder)
	return s.viewRecordsFunc().Read(s.ctx, vrkb.wsid, vrkb.IKeyBuilder, cb)
}
func (s *viewRecordsStorage) Validate([]ApplyBatchItem) (err error) { return err }
func (s *viewRecordsStorage) ApplyBatch(items []ApplyBatchItem) (err error) {
	batches := make(map[istructs.WSID][]istructs.ViewKV)
	nn := make(map[n10n]istructs.Offset)
	for _, item := range items {
		k := item.key.(*viewRecordsKeyBuilder)
		v := item.value.(*viewRecordsValueBuilder)
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
	return &viewRecordsValueBuilder{
		IValueBuilder: s.viewRecordsFunc().NewValueBuilder(kb.(*viewRecordsKeyBuilder).view),
		offset:        istructs.NullOffset,
		toJSONFunc:    s.toJSON,
		entity:        kb.Entity(),
	}
}
func (s *viewRecordsStorage) ProvideValueBuilderForUpdate(kb istructs.IStateKeyBuilder, existingValue istructs.IStateValue, _ istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	return &viewRecordsValueBuilder{
		IValueBuilder: s.viewRecordsFunc().UpdateValueBuilder(kb.(*viewRecordsKeyBuilder).view, existingValue.(*viewRecordsStorageValue).value),
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

	obj := make(map[string]interface{})
	s.schemasFunc().Schema(sv.AsQName(istructs.SystemField_QName)).
		Containers(func(containerName string, schema istructs.QName) {
			if containerName == istructs.SystemContainer_ViewValue {
				obj = coreutils.FieldsToMap(sv, s.schemasFunc(), coreutils.Filter(func(name string, kidn istructs.DataKindType) bool {
					return !options.excludedFields[name]
				}))
			}
		})

	bb, err := json.Marshal(obj)
	return string(bb), err
}
