/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"errors"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

type viewRecordsStorage struct {
	ctx             context.Context
	viewRecordsFunc viewRecordsFunc
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
		if errors.Is(err, istructsmem.ErrRecordNotFound) {
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
	var nullWsidBatch []istructs.ViewKV
	for wsid, batch := range batches {
		if wsid == istructs.NullWSID { // Actualizer offsets must be updated in the last order
			nullWsidBatch = batch
			continue
		}
		err = s.viewRecordsFunc().PutBatch(wsid, batch)
		if err != nil {
			return err
		}
	}
	if len(nullWsidBatch) > 0 {
		err = s.viewRecordsFunc().PutBatch(istructs.NullWSID, nullWsidBatch)
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
		entity:        kb.Entity(),
	}
}
func (s *viewRecordsStorage) ProvideValueBuilderForUpdate(kb istructs.IStateKeyBuilder, existingValue istructs.IStateValue, _ istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	return &viewValueBuilder{
		IValueBuilder: s.viewRecordsFunc().UpdateValueBuilder(kb.(*viewKeyBuilder).view, existingValue.(*viewValue).value),
		offset:        istructs.NullOffset,
		entity:        kb.Entity(),
	}
}
