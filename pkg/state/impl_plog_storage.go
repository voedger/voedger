/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"encoding/json"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type pLogStorage struct {
	ctx             context.Context
	eventsFunc      eventsFunc
	appDefFunc      appDefFunc
	partitionIDFunc PartitionIDFunc
}

func (s *pLogStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &pLogKeyBuilder{
		logKeyBuilder: logKeyBuilder{
			offset: istructs.FirstOffset,
			count:  1,
		},
		partitionID: s.partitionIDFunc(),
	}
}
func (s *pLogStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	err = s.Read(key, func(_ istructs.IKey, v istructs.IStateValue) (err error) {
		value = v
		return nil
	})
	return value, err
}
func (s *pLogStorage) Read(kb istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	k := kb.(*pLogKeyBuilder)
	cb := func(plogOffset istructs.Offset, event istructs.IPLogEvent) (err error) {
		offs := int64(plogOffset)
		return callback(
			&key{data: map[string]interface{}{Field_Offset: offs}},
			&pLogValue{
				event:      event,
				offset:     offs,
				toJSONFunc: s.toJSON,
			})
	}
	return s.eventsFunc().ReadPLog(s.ctx, k.partitionID, k.offset, k.count, cb)
}
func (s *pLogStorage) toJSON(sv istructs.IStateValue, _ ...interface{}) (string, error) {
	value := sv.(*pLogValue)
	obj := make(map[string]interface{})
	obj["QName"] = value.event.QName().String()
	obj["ArgumentObject"] = coreutils.ObjectToMap(value.event.ArgumentObject(), s.appDefFunc())
	cc := make([]map[string]interface{}, 0)
	value.event.CUDs(func(rec istructs.ICUDRow) {
		cudRowMap := cudRowToMap(rec, s.appDefFunc)
		cc = append(cc, cudRowMap)
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
