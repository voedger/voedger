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
	ctx              context.Context
	appStructsFunc   AppStructsFunc
	wsidFunc         WSIDFunc
	n10nFunc         N10nFunc
	wsTypeVailidator wsTypeVailidator
}

func newViewRecordsStorage(ctx context.Context, appStructsFunc AppStructsFunc, wsidFunc WSIDFunc, n10nFunc N10nFunc) *viewRecordsStorage {
	return &viewRecordsStorage{
		ctx:              ctx,
		appStructsFunc:   appStructsFunc,
		wsidFunc:         wsidFunc,
		n10nFunc:         n10nFunc,
		wsTypeVailidator: newWsTypeValidator(appStructsFunc),
	}
}

func (s *viewRecordsStorage) NewKeyBuilder(entity appdef.QName, _ istructs.IStateKeyBuilder) (newKeyBuilder istructs.IStateKeyBuilder) {
	return &viewKeyBuilder{
		IKeyBuilder: s.appStructsFunc().ViewRecords().KeyBuilder(entity),
		view:        entity,
		wsid:        s.wsidFunc(),
	}
}

func (s *viewRecordsStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	k := key.(*viewKeyBuilder)
	err = s.wsTypeVailidator.validate(k.wsid, k.view)
	if err != nil {
		return nil, err
	}
	v, err := s.appStructsFunc().ViewRecords().Get(k.wsid, k.IKeyBuilder)
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
		if err = s.wsTypeVailidator.validate(k.wsid, k.view); err != nil {
			return err
		}
		wsidToItemIdx[k.wsid] = append(wsidToItemIdx[k.wsid], itemIdx)
		batches[k.wsid] = append(batches[k.wsid], istructs.ViewRecordGetBatchItem{Key: k.IKeyBuilder})
	}
	for wsid, batch := range batches {
		err = s.appStructsFunc().ViewRecords().GetBatch(wsid, batch)
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
	k := kb.(*viewKeyBuilder)
	if err = s.wsTypeVailidator.validate(k.wsid, k.view); err != nil {
		return err
	}
	return s.appStructsFunc().ViewRecords().Read(s.ctx, k.wsid, k.IKeyBuilder, cb)
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
		err = s.appStructsFunc().ViewRecords().PutBatch(wsid, batch)
		if err != nil {
			return err
		}
	}
	if len(nullWsidBatch) > 0 {
		err = s.appStructsFunc().ViewRecords().PutBatch(istructs.NullWSID, nullWsidBatch)
		if err != nil {
			return err
		}
	}
	for n, newOffset := range nn {
		s.n10nFunc(n.view, n.wsid, newOffset)
	}
	return err
}
func (s *viewRecordsStorage) ProvideValueBuilder(kb istructs.IStateKeyBuilder, _ istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	k := kb.(*viewKeyBuilder)
	if err := s.wsTypeVailidator.validate(k.wsid, k.view); err != nil {
		return nil, err
	}
	return &viewValueBuilder{
		IValueBuilder: s.appStructsFunc().ViewRecords().NewValueBuilder(k.view),
		offset:        istructs.NullOffset,
		entity:        kb.Entity(),
	}, nil
}
func (s *viewRecordsStorage) ProvideValueBuilderForUpdate(kb istructs.IStateKeyBuilder, existingValue istructs.IStateValue, _ istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	k := kb.(*viewKeyBuilder)
	if err := s.wsTypeVailidator.validate(k.wsid, k.view); err != nil {
		return nil, err
	}
	return &viewValueBuilder{
		IValueBuilder: s.appStructsFunc().ViewRecords().UpdateValueBuilder(kb.(*viewKeyBuilder).view, existingValue.(*viewValue).value),
		offset:        istructs.NullOffset,
		entity:        kb.Entity(),
	}, nil
}
