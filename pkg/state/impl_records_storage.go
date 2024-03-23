/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type recordsStorage struct {
	recordsFunc      recordsFunc
	cudFunc          CUDFunc
	wsidFunc         WSIDFunc
	wsTypeVailidator wsTypeVailidator
}

func newRecordsStorage(appStructsFunc AppStructsFunc, wsidFunc WSIDFunc, cudFunc CUDFunc) *recordsStorage {
	return &recordsStorage{
		recordsFunc:      func() istructs.IRecords { return appStructsFunc().Records() },
		wsidFunc:         wsidFunc,
		cudFunc:          cudFunc,
		wsTypeVailidator: newWsTypeValidator(appStructsFunc),
	}
}

func (s *recordsStorage) NewKeyBuilder(entity appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &recordsKeyBuilder{
		id:        istructs.NullRecordID,
		singleton: appdef.NullQName,
		wsid:      s.wsidFunc(),
		entity:    entity,
	}
}

func (s *recordsStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	k := key.(*recordsKeyBuilder)
	if k.singleton != appdef.NullQName {
		err = s.wsTypeVailidator.validate(k.wsid, k.singleton)
		if err != nil {
			return nil, err
		}
		singleton, e := s.recordsFunc().GetSingleton(k.wsid, k.singleton)
		if e != nil {
			return nil, e
		}
		if singleton.QName() == appdef.NullQName {
			return nil, nil
		}
		return &recordsValue{record: singleton}, nil
	}
	if k.id == istructs.NullRecordID {
		// error message according to https://dev.untill.com/projects/#!637229
		return nil, fmt.Errorf("value of one of RecordID fields is 0: %w", ErrNotFound)
	}
	record, err := s.recordsFunc().Get(k.wsid, true, k.id)
	if err != nil {
		return nil, err
	}
	if record.QName() == appdef.NullQName {
		return nil, nil
	}
	return &recordsValue{record: record}, nil
}

func (s *recordsStorage) GetBatch(items []GetBatchItem) (err error) {
	type getSingletonParams struct {
		wsid    istructs.WSID
		qname   appdef.QName
		itemIdx int
	}
	wsidToItemIdx := make(map[istructs.WSID][]int)
	batches := make(map[istructs.WSID][]istructs.RecordGetBatchItem)
	gg := make([]getSingletonParams, 0)
	for itemIdx, item := range items {
		k := item.key.(*recordsKeyBuilder)
		if k.singleton != appdef.NullQName {
			err = s.wsTypeVailidator.validate(k.wsid, k.singleton)
			if err != nil {
				return err
			}
			gg = append(gg, getSingletonParams{
				wsid:    k.wsid,
				qname:   k.singleton,
				itemIdx: itemIdx,
			})
			continue
		}
		if k.id == istructs.NullRecordID {
			// error message according to https://dev.untill.com/projects/#!637229
			return fmt.Errorf("value of one of RecordID fields is 0: %w", ErrNotFound)
		}
		wsidToItemIdx[k.wsid] = append(wsidToItemIdx[k.wsid], itemIdx)
		batches[k.wsid] = append(batches[k.wsid], istructs.RecordGetBatchItem{ID: k.id})
	}
	for wsid, batch := range batches {
		err = s.recordsFunc().GetBatch(wsid, true, batch)
		if err != nil {
			return
		}
		for i, batchItem := range batch {
			if batchItem.Record.QName() == appdef.NullQName {
				continue
			}
			items[wsidToItemIdx[wsid][i]].value = &recordsValue{record: batchItem.Record}
		}
	}
	for _, g := range gg {
		singleton, e := s.recordsFunc().GetSingleton(g.wsid, g.qname)
		if e != nil {
			return e
		}
		if singleton.QName() == appdef.NullQName {
			continue
		}
		items[g.itemIdx].value = &recordsValue{record: singleton}
	}
	return err
}
func (s *recordsStorage) Validate([]ApplyBatchItem) (err error)   { return }
func (s *recordsStorage) ApplyBatch([]ApplyBatchItem) (err error) { return }
func (s *recordsStorage) ProvideValueBuilder(key istructs.IStateKeyBuilder, _ istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	kb := key.(*recordsKeyBuilder)
	if kb.entity == appdef.NullQName {
		return nil, errEntityRequiredForValueBuilder
	}
	err := s.wsTypeVailidator.validate(kb.wsid, kb.entity)
	if err != nil {
		return nil, err
	}
	rw := s.cudFunc().Create(kb.entity)
	return &recordsValueBuilder{rw: rw}, nil
}
func (s *recordsStorage) ProvideValueBuilderForUpdate(_ istructs.IStateKeyBuilder, existingValue istructs.IStateValue, _ istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	return &recordsValueBuilder{rw: s.cudFunc().Update(existingValue.AsRecord(""))}, nil
}
