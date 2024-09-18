/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package journal

import (
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
)

type predicate func(iws appdef.IWorkspace, qName appdef.QName) bool

type Filter struct {
	predicates []predicate
	iWorkspace appdef.IWorkspace
}

func NewFilter(iws appdef.IWorkspace, eventTypes []string, epJournalPredicates extensionpoints.IExtensionPoint) (Filter, error) {
	pp := make([]predicate, len(eventTypes))
	for i, eventType := range eventTypes {
		p, ok := epJournalPredicates.Find(eventType)
		if !ok {
			return Filter{}, fmt.Errorf("invalid event type: %s", eventType)
		}
		pp[i] = p.(func(iws appdef.IWorkspace, qName appdef.QName) bool)
	}
	return Filter{
		predicates: pp,
		iWorkspace: iws,
	}, nil
}

func (f Filter) isMatch(qName appdef.QName) bool {
	for _, p := range f.predicates {
		if p(f.iWorkspace, qName) {
			return true
		}
	}
	return false
}

type EventObject struct {
	istructs.NullObject
	Data  map[string]int64
	JSON  string
	Empty bool
}

func (o *EventObject) AsInt64(name string) int64 { return o.Data[name] }
func (o *EventObject) AsString(string) string    { return o.JSON }

func NewEventObject(event istructs.IWLogEvent, appDef appdef.IAppDef, f Filter, opts ...coreutils.MapperOpt) (o *EventObject, err error) {
	var bb []byte
	data := make(map[string]interface{})
	data["sys.QName"] = event.QName().String()
	data["RegisteredAt"] = event.RegisteredAt()
	data["Synced"] = event.Synced()
	data["DeviceID"] = event.DeviceID()
	data["SyncedAt"] = event.SyncedAt()
	noArgs := true
	if f.isMatch(event.ArgumentObject().QName()) {
		data["args"] = coreutils.ObjectToMap(event.ArgumentObject(), appDef, opts...)
		noArgs = false
	}
	cuds := coreutils.CUDsToMap(event, appDef, coreutils.WithFilter(f.isMatch), coreutils.WithMapperOpts(opts...))
	data["cuds"] = cuds
	bb, err = json.Marshal(&data)
	eo := &EventObject{
		Data:  make(map[string]int64),
		JSON:  string(bb),
		Empty: len(cuds) == 0 && noArgs,
	}
	eo.Data[Field_EventTime] = int64(event.RegisteredAt())
	return eo, err
}
