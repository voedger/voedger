/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"encoding/json"
	"fmt"

	"github.com/untillpro/voedger/pkg/istructs"
	coreutils "github.com/untillpro/voedger/pkg/utils"
)

type recordsStorage struct {
	recordsFunc recordsFunc
	cudFunc     CUDFunc
	schemasFunc schemasFunc
	wsidFunc    WSIDFunc
}

func (s *recordsStorage) NewKeyBuilder(entity istructs.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &recordsKeyBuilder{
		id:        istructs.NullRecordID,
		singleton: istructs.NullQName,
		wsid:      s.wsidFunc(),
		entity:    entity,
	}
}
func (s *recordsStorage) GetBatch(items []GetBatchItem) (err error) {
	type getSingletonParams struct {
		wsid    istructs.WSID
		qname   istructs.QName
		itemIdx int
	}
	wsidToItemIdx := make(map[istructs.WSID][]int)
	batches := make(map[istructs.WSID][]istructs.RecordGetBatchItem)
	gg := make([]getSingletonParams, 0)
	for itemIdx, item := range items {
		k := item.key.(*recordsKeyBuilder)
		if k.singleton != istructs.NullQName {
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
			if batchItem.Record.QName() == istructs.NullQName {
				continue
			}
			items[wsidToItemIdx[wsid][i]].value = &recordsStorageValue{
				record:     batchItem.Record,
				toJSONFunc: s.toJSON,
			}
		}
	}
	for _, g := range gg {
		singleton, e := s.recordsFunc().GetSingleton(g.wsid, g.qname)
		if e != nil {
			return e
		}
		if singleton.QName() == istructs.NullQName {
			continue
		}
		items[g.itemIdx].value = &recordsStorageValue{
			record:     singleton,
			toJSONFunc: s.toJSON,
		}
	}
	return err
}
func (s *recordsStorage) Validate([]ApplyBatchItem) (err error)   { return }
func (s *recordsStorage) ApplyBatch([]ApplyBatchItem) (err error) { return }
func (s *recordsStorage) ProvideValueBuilder(key istructs.IStateKeyBuilder, _ istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	rw := s.cudFunc().Create(key.(*recordsKeyBuilder).entity)
	return &recordsValueBuilder{rw: rw}
}
func (s *recordsStorage) ProvideValueBuilderForUpdate(_ istructs.IStateKeyBuilder, existingValue istructs.IStateValue, _ istructs.IStateValueBuilder) istructs.IStateValueBuilder {
	return &recordsValueBuilder{rw: s.cudFunc().Update(existingValue.AsRecord(""))}
}
func (s *recordsStorage) toJSON(sv istructs.IStateValue, _ ...interface{}) (string, error) {
	obj := coreutils.FieldsToMap(sv, s.schemasFunc())
	bb, err := json.Marshal(&obj)
	return string(bb), err
}
