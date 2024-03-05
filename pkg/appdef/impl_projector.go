/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// # Implements:
//   - IProjector & IProjectorBuilder
type projector struct {
	extension
	sync      bool
	events    projectorEvents
	eventsMap map[QName][]ProjectorEventKind
	sysErrors bool
	states    *storages
	intents   *storages
}

func newProjector(app *appDef, name QName) *projector {
	prj := &projector{
		events:    make(projectorEvents),
		eventsMap: make(map[QName][]ProjectorEventKind),
		states:    newStorages(),
		intents:   newStorages(),
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
	case TypeKind_GDoc, TypeKind_GRecord, TypeKind_CDoc, TypeKind_CRecord, TypeKind_WDoc, TypeKind_WRecord, // CUD
		TypeKind_Command,               // Execute
		TypeKind_ODoc, TypeKind_Object: // Execute with
		if e, ok := prj.events[on]; ok {
			e.addKind(event...)
			prj.eventsMap[on] = e.Kind()
		} else {
			e := newProjectorEvent(t, event...)
			prj.events[on] = e
			prj.eventsMap[on] = e.Kind()
		}
	default:
		panic(fmt.Errorf("%v: %v is not applicable for projector event: %w", prj, t, ErrInvalidProjectorEventKind))
	}
	return prj
}

func (prj *projector) Events(cb func(IProjectorEvent)) {
	ord := QNamesFromMap(prj.events)
	for _, n := range ord {
		cb(prj.events[n])
	}
}

func (prj *projector) EventsMap() map[QName][]ProjectorEventKind {
	return prj.eventsMap
}

func (prj *projector) Intents() IStorages {
	return prj.intents
}

func (prj *projector) IntentsBuilder() IStoragesBuilder {
	return prj.intents
}

func (prj *projector) SetEventComment(record QName, comment ...string) IProjectorBuilder {
	e, ok := prj.events[record]
	if !ok {
		panic(fmt.Errorf("%v: %v not found: %w", prj, record, ErrNameNotFound))
	}
	e.SetComment(comment...)
	return prj
}

func (prj *projector) States() IStorages {
	return prj.states
}

func (prj *projector) StatesBuilder() IStoragesBuilder {
	return prj.states
}

func (prj *projector) SetSync(sync bool) IProjectorBuilder {
	prj.sync = sync
	return prj
}

func (prj *projector) SetWantErrors() IProjectorBuilder {
	prj.sysErrors = true
	return prj
}

func (prj *projector) Sync() bool { return prj.sync }

func (prj *projector) WantErrors() bool { return prj.sysErrors }

// Validates projector
//
// # Returns error:
//   - if events set is empty
func (prj *projector) Validate() (err error) {
	if len(prj.events) == 0 {
		err = errors.Join(err,
			fmt.Errorf("%v: events set is empty: %w", prj, ErrEmptyProjectorEvents))
	}
	return err
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

func (i ProjectorEventKind) MarshalText() ([]byte, error) {
	var s string
	if (i > 0) && (i < ProjectorEventKind_Count) {
		s = i.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(i), base)
	}
	return []byte(s), nil
}

// Renders an ProjectorEventKind in human-readable form, without `ProjectorEventKind_` prefix,
// suitable for debugging or error messages
func (i ProjectorEventKind) TrimString() string {
	const pref = "ProjectorEventKind_"
	return strings.TrimPrefix(i.String(), pref)
}

// Returns is event kind compatible with type kind.
//
// # Compatibles:
//
//   - Any document or record can be inserted.
//   - Any document or record, except ODoc and ORecord, can be updated, activated or deactivated.
//   - Only command can be executed.
//   - Only object or ODoc can be parameter for command execute with.
func (i ProjectorEventKind) typeCompatible(kind TypeKind) bool {
	switch i {
	case ProjectorEventKind_Insert, ProjectorEventKind_Update, ProjectorEventKind_Activate, ProjectorEventKind_Deactivate:
		return kind == TypeKind_GDoc || kind == TypeKind_GRecord ||
			kind == TypeKind_CDoc || kind == TypeKind_CRecord ||
			kind == TypeKind_WDoc || kind == TypeKind_WRecord
	case ProjectorEventKind_Execute:
		return kind == TypeKind_Command
	case ProjectorEventKind_ExecuteWithParam:
		return kind == TypeKind_Object || kind == TypeKind_ODoc
	}
	return false
}
