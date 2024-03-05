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
//   - IProjector
type projector struct {
	extension
	sync      bool
	sysErrors bool
	events    *projectorEvents
	states    *storages
	intents   *storages
}

func newProjector(app *appDef, name QName) *projector {
	prj := &projector{
		events:  newProjectorEvents(app),
		states:  newStorages(),
		intents: newStorages(),
	}
	prj.extension = makeExtension(app, name, TypeKind_Projector, prj)
	app.appendType(prj)
	return prj
}

func (prj *projector) Events() IProjectorEvents { return prj.events }

func (prj *projector) Intents() IStorages { return prj.intents }

func (prj *projector) States() IStorages { return prj.states }

func (prj *projector) Sync() bool { return prj.sync }

func (prj *projector) WantErrors() bool { return prj.sysErrors }

// # Implements:
//   - IProjectorBuilder
type projectorBuilder struct {
	*projector
}

func newProjectorBuilder(prj *projector) *projectorBuilder {
	return &projectorBuilder{prj}
}

func (prj *projectorBuilder) Events() IProjectorEventsBuilder { return prj.projector.events }

func (prj *projectorBuilder) Intents() IStoragesBuilder { return prj.projector.intents }

func (prj *projectorBuilder) States() IStoragesBuilder { return prj.projector.states }

func (prj *projectorBuilder) SetSync(sync bool) IProjectorBuilder {
	prj.projector.sync = sync
	return prj
}

func (prj *projectorBuilder) SetWantErrors() IProjectorBuilder {
	prj.projector.sysErrors = true
	return prj
}

// Validates projector
//
// # Returns error:
//   - if events set is empty
func (prj *projector) Validate() (err error) {
	if len(prj.events.events) == 0 {
		err = errors.Join(err,
			fmt.Errorf("%v: events set is empty: %w", prj, ErrEmptyProjectorEvents))
	}
	return err
}

// # Implements:
//   - IProjectorEvents
//   - IProjectorEventsBuilder
type projectorEvents struct {
	app       *appDef
	events    map[QName]*projectorEvent
	eventsMap map[QName][]ProjectorEventKind
}

func newProjectorEvents(app *appDef) *projectorEvents {
	return &projectorEvents{
		app:       app,
		events:    make(map[QName]*projectorEvent),
		eventsMap: make(map[QName][]ProjectorEventKind),
	}
}

func (ee *projectorEvents) Add(on QName, event ...ProjectorEventKind) IProjectorEventsBuilder {
	if on == NullQName {
		panic(ErrNameMissed)
	}

	t := ee.app.TypeByName(on)
	if t == nil {
		panic(fmt.Errorf("type «%v» not found: %w", on, ErrNameNotFound))
	}
	switch t.Kind() {
	case TypeKind_GDoc, TypeKind_GRecord, TypeKind_CDoc, TypeKind_CRecord, TypeKind_WDoc, TypeKind_WRecord, // CUD
		TypeKind_Command,               // Execute
		TypeKind_ODoc, TypeKind_Object: // Execute with
		e, ok := ee.events[on]
		if ok {
			e.addKind(event...)
		} else {
			e = newProjectorEvent(t, event...)
			ee.events[on] = e
		}
		ee.eventsMap[on] = e.Kind()
	default:
		panic(fmt.Errorf("%v is not applicable for projector event: %w", t, ErrInvalidProjectorEventKind))
	}
	return ee
}

func (ee projectorEvents) Enum(cb func(IProjectorEvent)) {
	ord := QNamesFromMap(ee.events)
	for _, n := range ord {
		cb(ee.events[n])
	}
}

func (ee projectorEvents) Event(name QName) IProjectorEvent {
	return ee.events[name]
}

func (ee projectorEvents) Len() int {
	return len(ee.events)
}

func (ee projectorEvents) Map() map[QName][]ProjectorEventKind {
	return ee.eventsMap
}

func (ee *projectorEvents) SetComment(record QName, comment ...string) IProjectorEventsBuilder {
	e, ok := ee.events[record]
	if !ok {
		panic(ErrNameNotFound)
	}
	e.SetComment(comment...)
	return ee
}

// # Implements:
//   - IProjectorEvent
type projectorEvent struct {
	comment
	on    IType
	kinds uint64 // bitmap[ProjectorEventKind]
}

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
