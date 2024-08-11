/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

type wLogStorage struct {
	ctx        context.Context
	eventsFunc eventsFunc
	wsidFunc   WSIDFunc
}

type wLogKeyBuilder struct {
	baseKeyBuilder
	offset istructs.Offset
	count  int
	wsid   istructs.WSID
}

func (b *wLogKeyBuilder) Storage() appdef.QName {
	return sys.Storage_WLog
}

func (b *wLogKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*wLogKeyBuilder)
	if !ok {
		return false
	}
	kb := src.(*wLogKeyBuilder)
	if kb.count != b.count {
		return false
	}
	if kb.offset != b.offset {
		return false
	}
	if kb.wsid != b.wsid {
		return false
	}
	return true
}

func (b *wLogKeyBuilder) String() string {
	return fmt.Sprintf("wlog wsid - %d, offset - %d, count - %d", b.wsid, b.offset, b.count)
}

func (b *wLogKeyBuilder) PutInt64(name string, value int64) {
	if name == sys.Storage_WLog_Field_WSID {
		b.wsid = istructs.WSID(value)
	} else if name == sys.Storage_WLog_Field_Offset {
		b.offset = istructs.Offset(value)
	} else if name == sys.Storage_WLog_Field_Count {
		b.count = int(value)
	} else {
		b.baseKeyBuilder.PutInt64(name, value)
	}
}

func (s *wLogStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &wLogKeyBuilder{
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
			&key{data: map[string]interface{}{sys.Storage_WLog_Field_Offset: offs}},
			&wLogValue{
				event:  event,
				offset: offs,
			})
	}
	return s.eventsFunc().ReadWLog(s.ctx, k.wsid, k.offset, k.count, cb)
}

type wLogValue struct {
	baseStateValue
	event  istructs.IWLogEvent
	offset int64
}

func (v *wLogValue) AsInt64(name string) int64 {
	switch name {
	case sys.Storage_WLog_Field_RegisteredAt:
		return int64(v.event.RegisteredAt())
	case sys.Storage_WLog_Field_DeviceID:
		return int64(v.event.DeviceID())
	case sys.Storage_WLog_Field_SyncedAt:
		return int64(v.event.SyncedAt())
	case sys.Storage_WLog_Field_Offset:
		return v.offset
	default:
		return v.baseStateValue.AsInt64(name)
	}
}
func (v *wLogValue) AsBool(_ string) bool          { return v.event.Synced() }
func (v *wLogValue) AsQName(_ string) appdef.QName { return v.event.QName() }
func (v *wLogValue) AsEvent(_ string) (event istructs.IDbEvent) {
	return v.event
}
func (v *wLogValue) AsRecord(_ string) (record istructs.IRecord) {
	return v.event.ArgumentObject().AsRecord()
}
func (v *wLogValue) AsValue(name string) istructs.IStateValue {
	if name == sys.Storage_WLog_Field_CUDs {
		sv := &cudsValue{}
		v.event.CUDs(func(rec istructs.ICUDRow) {
			sv.cuds = append(sv.cuds, rec)
		})
		return sv
	}
	if name == sys.Storage_WLog_Field_ArgumentObject {
		arg := v.event.ArgumentObject()
		if arg == nil {
			return nil
		}
		return &objectValue{object: arg}
	}
	return v.baseStateValue.AsValue(name)
}
