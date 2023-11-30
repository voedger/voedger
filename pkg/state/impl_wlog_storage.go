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

type wLogStorage struct {
	ctx        context.Context
	eventsFunc eventsFunc
	appDefFunc appDefFunc
	wsidFunc   WSIDFunc
}

func (s *wLogStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &wLogKeyBuilder{
		logKeyBuilder: logKeyBuilder{
			offset: istructs.FirstOffset,
			count:  1,
		},
		wsid: s.wsidFunc(),
	}
}
func (s *wLogStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	err = s.Read(key, func(_ istructs.IKey, v istructs.IStateValue) (err error) {
		value = v
		return nil
	})
	return value, err
}
func (s *wLogStorage) Read(kb istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	k := kb.(*wLogKeyBuilder)
	cb := func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
		offs := int64(wlogOffset)
		return callback(
			&key{data: map[string]interface{}{Field_Offset: offs}},
			&wLogValue{
				event:      event,
				offset:     offs,
				toJSONFunc: s.toJSON,
			})
	}
	return s.eventsFunc().ReadWLog(s.ctx, k.wsid, k.offset, k.count, cb)
}
func (s *wLogStorage) toJSON(sv istructs.IStateValue, _ ...interface{}) (string, error) {
	value := sv.(*wLogValue)
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
