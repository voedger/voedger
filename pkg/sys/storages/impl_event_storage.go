/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

type eventStorage struct {
	eventFunc PLogEventFunc
}

type eventKeyBuilder struct {
	baseKeyBuilder
}

func (b *eventKeyBuilder) Storage() appdef.QName {
	return sys.Storage_Event
}

func (b *eventKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*eventKeyBuilder)
	return ok
}

type eventValue struct {
	baseStateValue
	event  istructs.IPLogEvent
	offset int64
}

func (v *eventValue) AsInt64(name string) int64 {
	switch name {
	case sys.Storage_Event_Field_WLogOffset:
		return int64(v.event.WLogOffset())
	case sys.Storage_Event_Field_Workspace:
		return int64(v.event.Workspace())
	case sys.Storage_Event_Field_RegisteredAt:
		return int64(v.event.RegisteredAt())
	case sys.Storage_Event_Field_DeviceID:
		return int64(v.event.DeviceID())
	case sys.Storage_Event_Field_SyncedAt:
		return int64(v.event.SyncedAt())
	case sys.Storage_Event_Field_Offset:
		return v.offset
	}
	return v.baseStateValue.AsInt64(name)
}
func (v *eventValue) AsBool(name string) bool {
	if name == sys.Storage_Event_Field_Synced {
		return v.event.Synced()
	}
	return v.baseStateValue.AsBool(name)
}
func (v *eventValue) AsRecord(string) istructs.IRecord {
	return v.event.ArgumentObject().AsRecord()
}
func (v *eventValue) AsQName(name string) appdef.QName {
	if name == sys.Storage_Event_Field_QName {
		return v.event.QName()
	}
	return v.baseStateValue.AsQName(name)
}
func (v *eventValue) AsEvent(string) istructs.IDbEvent { return v.event }
func (v *eventValue) AsValue(name string) istructs.IStateValue {
	if name == sys.Storage_Event_Field_CUDs {
		sv := &cudsValue{}
		v.event.CUDs(func(rec istructs.ICUDRow) {
			sv.cuds = append(sv.cuds, rec)
		})
		return sv
	}
	if name == sys.Storage_Event_Field_Error {
		return &eventErrorValue{error: v.event.Error()}
	}
	if name == sys.Storage_Event_Field_ArgumentObject {
		arg := v.event.ArgumentObject()
		if arg == nil {
			return nil
		}
		return &objectValue{object: arg}
	}
	return v.baseStateValue.AsValue(name)
}

func newEventStorage(eventFunc PLogEventFunc) *eventStorage {
	return &eventStorage{
		eventFunc: eventFunc,
	}
}

func (s *eventStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &eventKeyBuilder{}
}
func (s *eventStorage) Get(_ istructs.IStateKeyBuilder) (istructs.IStateValue, error) {
	return &eventValue{
		event: s.eventFunc(),
	}, nil
}
