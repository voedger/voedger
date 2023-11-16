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
	typ
	sync    bool
	ext     *extension
	events  projectorEvents
	states  storages
	intents storages
}

func newProjector(app *appDef, name QName) *projector {
	prj := &projector{
		typ:     makeType(app, name, TypeKind_Projector),
		ext:     newExtension(),
		events:  make(projectorEvents),
		states:  make(storages),
		intents: make(storages),
	}
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
	case TypeKind_GDoc, TypeKind_GRecord,
		TypeKind_CDoc, TypeKind_CRecord,
		TypeKind_WDoc, TypeKind_WRecord,
		TypeKind_ODoc, TypeKind_ORecord,
		TypeKind_Command:
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

func (prj *projector) Extension() IExtension { return prj.ext }

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

func (prj *projector) SetExtension(name string, engine ExtensionEngineKind, comment ...string) IProjectorBuilder {
	if name == "" {
		name = prj.QName().Entity()
	}
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: extension name «%s» is not valid: %w", prj, name, err))
	}
	prj.ext.name = name
	prj.ext.engine = engine
	prj.ext.SetComment(comment...)

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
	switch on.Kind() {
	case TypeKind_Command:
		p.addKind(ProjectorEventKind_Execute)
	}
	p.addKind(kind...)
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
