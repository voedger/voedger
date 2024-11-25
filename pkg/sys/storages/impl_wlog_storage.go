/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"bytes"
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

type wLogStorage struct {
	ctx        context.Context
	eventsFunc state.EventsFunc
	wsidFunc   state.WSIDFunc
}

func NewWLogStorage(ctx context.Context, eventsFunc state.EventsFunc, wsidFunc state.WSIDFunc) state.IStateStorage {
	return &wLogStorage{
		ctx:        ctx,
		eventsFunc: eventsFunc,
		wsidFunc:   wsidFunc,
	}
}

type wLogKeyBuilder struct {
	baseKeyBuilder
	offset istructs.Offset
	count  int
	wsid   istructs.WSID
}

func (b *wLogKeyBuilder) String() string {
	bb := new(bytes.Buffer)
	fmt.Fprint(bb, b.baseKeyBuilder.String())
	fmt.Fprintf(bb, ", wsid:%d", b.wsid)
	fmt.Fprintf(bb, ", offset:%d", b.offset)
	fmt.Fprintf(bb, ", count:%d", b.count)
	return bb.String()
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

func (b *wLogKeyBuilder) PutInt64(name string, value int64) {
	if name == sys.Storage_WLog_Field_WSID {
		wsid, err := coreutils.Int64ToWSID(value)
		if err != nil {
			panic(err)
		}
		b.wsid = wsid
	} else if name == sys.Storage_WLog_Field_Offset {
		b.offset = istructs.Offset(value) // nolint G115
	} else if name == sys.Storage_WLog_Field_Count {
		b.count = int(value)
	} else {
		b.baseKeyBuilder.PutInt64(name, value)
	}
}

func (s *wLogStorage) NewKeyBuilder(appdef.QName, istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &wLogKeyBuilder{
		baseKeyBuilder: baseKeyBuilder{storage: sys.Storage_WLog},
		wsid:           s.wsidFunc(),
	}
}
func (s *wLogStorage) Get(kb istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	k := kb.(*wLogKeyBuilder)
	cb := func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
		value = &wLogValue{
			event:  event,
			offset: wlogOffset,
		}
		return nil
	}
	err = s.eventsFunc().ReadWLog(s.ctx, k.wsid, k.offset, 1, cb)
	return value, err
}
func (s *wLogStorage) Read(kb istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	k := kb.(*wLogKeyBuilder)
	cb := func(wlogOffset istructs.Offset, event istructs.IWLogEvent) (err error) {
		offs := wlogOffset
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
	istructs.IStateWLogValue
	event  istructs.IWLogEvent
	offset istructs.Offset
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
		return int64(v.offset) // nolint G115
	default:
		return v.baseStateValue.AsInt64(name)
	}
}
func (v *wLogValue) AsBool(_ string) bool          { return v.event.Synced() }
func (v *wLogValue) AsQName(_ string) appdef.QName { return v.event.QName() }
func (v *wLogValue) AsEvent() (event istructs.IWLogEvent) {
	return v.event
}
func (v *wLogValue) AsValue(name string) istructs.IStateValue {
	if name == sys.Storage_WLog_Field_CUDs {
		sv := &cudsValue{}
		for rec := range v.event.CUDs {
			sv.cuds = append(sv.cuds, rec)
		}
		return sv
	}
	if name == sys.Storage_WLog_Field_ArgumentObject {
		arg := v.event.ArgumentObject()
		if arg == nil {
			return nil
		}
		return &ObjectStateValue{object: arg}
	}
	return v.baseStateValue.AsValue(name)
}

type key struct {
	istructs.IKey
	data map[string]interface{}
}

func (k *key) AsInt64(name string) int64 { return int64(k.data[name].(istructs.Offset)) } // nolint G115
