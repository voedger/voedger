/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package journal

import (
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/vvm"
)

type predicate func(appDef appdef.IAppDef, qName appdef.QName) bool

type Filter struct {
	predicates []predicate
	appDef     appdef.IAppDef
}

func NewFilter(appDef appdef.IAppDef, eventTypes []string, jp vvm.IEPJournalPredicates) (Filter, error) {
	pp := make([]predicate, len(eventTypes))
	for i, eventType := range eventTypes {
		p, ok := jp.Find(eventType)
		if !ok {
			return Filter{}, fmt.Errorf("invalid event type: %s", eventType)
		}
		pp[i] = p.(func(appDef appdef.IAppDef, qName appdef.QName) bool)
	}
	return Filter{
		predicates: pp,
		appDef:     appDef,
	}, nil
}

func (f Filter) isMatch(qName appdef.QName) bool {
	for _, p := range f.predicates {
		if p(f.appDef, qName) {
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
	cuds := make([]map[string]interface{}, 0)
	err = event.CUDs(func(rec istructs.ICUDRow) error {
		if !f.isMatch(rec.QName()) {
			return err
		}
		cud := make(map[string]interface{})
		cud["sys.ID"] = rec.ID()
		cud["sys.QName"] = rec.QName().String()
		cud["IsNew"] = rec.IsNew()
		cud["fields"] = coreutils.FieldsToMap(rec, appDef, opts...)
		cuds = append(cuds, cud)
		return nil
	})
	if err != nil {
		// notest
		return nil, err
	}
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
