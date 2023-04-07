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

type pLogStorage struct {
	ctx             context.Context
	eventsFunc      eventsFunc
	schemasFunc     schemasFunc
	partitionIDFunc PartitionIDFunc
}

func (s *pLogStorage) NewKeyBuilder(istructs.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &pLogKeyBuilder{
		logKeyBuilder: logKeyBuilder{
			offset: istructs.FirstOffset,
			count:  1,
		},
		partitionID: s.partitionIDFunc(),
	}
}
func (s *pLogStorage) GetBatch(items []GetBatchItem) (err error) {
	for i := range items {
		skip := false
		err = s.Read(items[i].key, func(_ istructs.IKey, value istructs.IStateValue) (err error) {
			if skip {
				return
			}
			items[i].value = value
			skip = true
			return
		})
		if err != nil {
			break
		}
	}
	return err
}
func (s *pLogStorage) Read(kb istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	k := kb.(*pLogKeyBuilder)
	cb := func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
		offs := int64(plogOffset)
		return callback(
			&key{data: map[string]interface{}{Field_Offset: offs}},
			&pLogStorageValue{
				event:      event,
				offset:     offs,
				toJSONFunc: s.toJSON,
			})
	}
	return s.eventsFunc().ReadPLog(s.ctx, k.partitionID, k.offset, k.count, cb)
}
func (s *pLogStorage) toJSON(sv istructs.IStateValue, _ ...interface{}) (string, error) {
	value := sv.(*pLogStorageValue)
	obj := make(map[string]interface{})
	obj["QName"] = value.event.QName().String()
	obj["ArgumentObject"] = coreutils.ObjectToMap(value.event.ArgumentObject(), s.schemasFunc())
	cc := make([]map[string]interface{}, 0)
	_ = value.event.CUDs(func(rec istructs.ICUDRow) (err error) { //no error returns
		cudRowMap := cudRowToMap(rec, s.schemasFunc)
		cc = append(cc, cudRowMap)
		return
	})
	obj["CUDs"] = cc
	obj[Field_RegisteredAt] = value.event.RegisteredAt()
	obj["Synced"] = value.event.Synced()
	obj[Field_DeviceID] = value.event.DeviceID()
	obj[Field_SyncedAt] = value.event.SyncedAt()
	obj[Field_Workspace] = value.event.Workspace()
	obj[Field_WLogOffset] = value.event.WLogOffset()
	errObj := make(map[string]interface{})
	errObj["ErrStr"] = value.event.Error().ErrStr()
	errObj["QNameFromParams"] = value.event.Error().QNameFromParams().String()
	errObj["ValidEvent"] = value.event.Error().ValidEvent()
	errObj["OriginalEventBytes"] = value.event.Error().OriginalEventBytes()
	obj["Error"] = errObj
	obj[Field_Offset] = value.offset
	bb, err := json.Marshal(&obj)
	return string(bb), err
}
