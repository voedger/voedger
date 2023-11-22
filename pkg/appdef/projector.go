/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"strings"
)

// # Implements:
//   - IProjector & IProjectorBuilder
type projector struct {
	extension
	sync    bool
	events  projectorEvents
	states  storages
	intents storages
}

func newProjector(app *appDef, name QName) *projector {
	prj := &projector{
		events:  make(projectorEvents),
		states:  make(storages),
		intents: make(storages),
	}
	prj.extension = makeExtension(app, name, TypeKind_Projector, prj)
	app.appendType(prj)
	return prj
}

func (prj *projector) AddEvent(on QName, event ...ProjectorEventKind) IProjectorBuilder {
	if on == NullQName {
		panic(fmt.Errorf("%v: type name is empty: %w", prj, ErrNameMissed))
	}

	t := prj.app.TypeByName(on)
	if t == nil {
		panic(fmt.Errorf("%v: type «%v» not found: %w", prj, on, ErrNameNotFound))
	}
	switch t.Kind() {
	case TypeKind_GDoc, TypeKind_GRecord, TypeKind_CDoc, TypeKind_CRecord, TypeKind_WDoc, TypeKind_WRecord,
		TypeKind_Command,
		TypeKind_ODoc, TypeKind_Object:
		if e, ok := prj.events[on]; ok {
			e.addKind(event...)
		} else {
			prj.events[on] = newProjectorEvent(t, event...)
		}
	default:
		panic(fmt.Errorf("%v: %v is not applicable for projector event: %w", prj, t, ErrInvalidProjectorEventKind))
	}
	return prj
}

func (prj *projector) AddState(storage QName, names ...QName) IProjectorBuilder {
	prj.states.add(storage, names...)
	return prj
}

func (prj *projector) AddIntent(storage QName, names ...QName) IProjectorBuilder {
	prj.intents.add(storage, names...)
	return prj
}

func (prj *projector) Events(cb func(IProjectorEvent)) {
	ord := QNamesFromMap(prj.events)
	for _, n := range ord {
		cb(prj.events[n])
	}
}

func (prj *projector) Intents(cb func(storage QName, names QNames)) {
	prj.intents.enum(cb)
}

func (prj *projector) SetEventComment(record QName, comment ...string) IProjectorBuilder {
	e, ok := prj.events[record]
	if !ok {
		panic(fmt.Errorf("%v: %v not found: %w", prj, record, ErrNameNotFound))
	}
	e.SetComment(comment...)
	return prj
}

func (prj *projector) SetSync(sync bool) IProjectorBuilder {
	prj.sync = sync
	return prj
}

func (prj *projector) Sync() bool { return prj.sync }

func (prj *projector) States(cb func(storage QName, names QNames)) {
	prj.states.enum(cb)
}

type (
	// # Implements:
	//	 - IProjectorEvent
	projectorEvent struct {
		comment
		on    IType
		kinds uint64 // bitmap[ProjectorEventKind]
	}
	projectorEvents map[QName]*projectorEvent
)

func newProjectorEvent(on IType, kind ...ProjectorEventKind) *projectorEvent {
	p := &projectorEvent{on: on}

	if len(kind) > 0 {
		p.addKind(kind...)
	} else {
		// missed kind, make defaults
		switch on.Kind() {
		case TypeKind_GDoc, TypeKind_GRecord, TypeKind_CDoc, TypeKind_CRecord, TypeKind_WDoc, TypeKind_WRecord:
			p.addKind(ProjectorEventKind_AnyChanges...)
		case TypeKind_Command:
			p.addKind(ProjectorEventKind_Execute)
		case TypeKind_Object, TypeKind_ODoc:
			p.addKind(ProjectorEventKind_ExecuteWithParam)
		}
	}

	return p
}

func (e *projectorEvent) Kind() (kinds []ProjectorEventKind) {
	for k := ProjectorEventKind(1); k < ProjectorEventKind_Count; k++ {
		if e.kinds&(1<<k) != 0 {
			kinds = append(kinds, k)
		}
	}
	return kinds
}

func (e *projectorEvent) On() IType {
	return e.on
}

func (e projectorEvent) String() string {
	s := []string{}
	for _, k := range e.Kind() {
		s = append(s, k.TrimString())
	}
	return fmt.Sprintf("%v [%s]", e.On(), strings.Join(s, " "))
}

// Adds specified events to projector event.
//
// # Panics:
//   - if event kind is not compatible with type.
func (e *projectorEvent) addKind(kind ...ProjectorEventKind) {
	for _, k := range kind {
		if !k.typeCompatible(e.on.Kind()) {
			panic(fmt.Errorf("%s event is not applicable with %v: %w", k.TrimString(), e.on, ErrInvalidProjectorEventKind))
		}
		e.kinds |= 1 << k
	}
}

type storages map[QName]QNames

func (ss storages) add(name QName, names ...QName) {
	q, ok := ss[name]
	if !ok {
		q = QNames{}
	}
	q.Add(names...)
	ss[name] = q
}

func (ss storages) enum(cb func(storage QName, names QNames)) {
	ord := QNamesFromMap(ss)
	for _, n := range ord {
		cb(n, ss[n])
	}
}
