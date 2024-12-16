/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"strings"

	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/set"
)

// # Implements:
//   - IProjector
type projector struct {
	extension
	sync      bool
	sysErrors bool
	events    *events
}

func newProjector(app *appDef, ws *workspace, name QName) *projector {
	prj := &projector{
		extension: makeExtension(app, ws, name, TypeKind_Projector),
		events:    newProjectorEvents(app),
	}
	ws.appendType(prj)
	return prj
}

func (prj projector) Events() IProjectorEvents { return prj.events }

func (prj projector) Sync() bool { return prj.sync }

func (prj projector) WantErrors() bool { return prj.sysErrors }

func (prj *projector) setSync(sync bool) { prj.sync = sync }

func (prj *projector) setWantErrors() { prj.sysErrors = true }

// # Implements:
//   - IProjectorBuilder
type projectorBuilder struct {
	extensionBuilder
	*projector
	events *eventsBuilder
}

func newProjectorBuilder(projector *projector) *projectorBuilder {
	return &projectorBuilder{
		extensionBuilder: makeExtensionBuilder(&projector.extension),
		projector:        projector,
		events:           newEventsBuilder(projector.events),
	}
}

func (pb *projectorBuilder) Events() IProjectorEventsBuilder { return pb.events }

func (pb *projectorBuilder) SetSync(sync bool) IProjectorBuilder {
	pb.projector.setSync(sync)
	return pb
}

func (pb *projectorBuilder) SetWantErrors() IProjectorBuilder {
	pb.projector.setWantErrors()
	return pb
}

// Validates projector
//
// # Returns error:
//   - if events set is empty
func (prj *projector) Validate() (err error) {
	err = prj.extension.Validate()

	if len(prj.events.events) == 0 {
		err = errors.Join(err, ErrMissed("%v events", prj))
	}

	return err
}

// # Implements:
//   - IProjectorEvents
type events struct {
	app       *appDef
	events    map[QName]*event
	eventsMap map[QName][]ProjectorEventKind
}

func newProjectorEvents(app *appDef) *events {
	return &events{
		app:       app,
		events:    make(map[QName]*event),
		eventsMap: make(map[QName][]ProjectorEventKind),
	}
}

func (ee events) Enum(cb func(IProjectorEvent)) {
	ord := QNamesFromMap(ee.events)
	for _, n := range ord {
		cb(ee.events[n])
	}
}

func (ee events) Event(name QName) IProjectorEvent {
	return ee.events[name]
}

func (ee events) Len() int {
	return len(ee.events)
}

func (ee events) Map() map[QName][]ProjectorEventKind {
	return ee.eventsMap
}

func (ee *events) add(on QName, event ...ProjectorEventKind) {
	if on == NullQName {
		panic(ErrMissed("event name"))
	}

	t := ee.app.Type(on)
	if t.Kind() == TypeKind_null {
		panic(ErrTypeNotFound(on))
	}

	e, ok := ee.events[on]
	if ok {
		e.addKind(event...)
	} else {
		e = newEvent(t, event...)
		ee.events[on] = e
	}
	ee.eventsMap[on] = e.Kind()
}

func (ee *events) setComment(on QName, comment ...string) {
	e, ok := ee.events[on]
	if !ok {
		panic(ErrNotFound("event name «%v»", on))
	}
	e.comment.setComment(comment...)
}

// # Implements:
//   - IProjectorEventsBuilder
type eventsBuilder struct {
	*events
}

func newEventsBuilder(events *events) *eventsBuilder {
	return &eventsBuilder{
		events: events,
	}
}

func (eb *eventsBuilder) Add(on QName, event ...ProjectorEventKind) IProjectorEventsBuilder {
	eb.events.add(on, event...)
	return eb
}

func (eb *eventsBuilder) SetComment(record QName, comment ...string) IProjectorEventsBuilder {
	eb.events.setComment(record, comment...)
	return eb
}

// # Implements:
//   - IProjectorEvent
type event struct {
	comment
	on    IType
	kinds set.Set[ProjectorEventKind]
}

func newEvent(on IType, kind ...ProjectorEventKind) *event {
	p := &event{on: on}

	if len(kind) > 0 {
		p.addKind(kind...)
	} else {
		kinds := allProjectorEventsOnType(on)
		if kinds.Len() == 0 {
			panic(ErrUnsupported("type %v can't be projector trigger", on))
		}
		p.addKind(kinds.AsArray()...)
	}

	return p
}

func (e *event) Kind() (kinds []ProjectorEventKind) {
	return e.kinds.AsArray()
}

func (e *event) On() IType {
	return e.on
}

func (e event) String() string {
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
func (e *event) addKind(kind ...ProjectorEventKind) {
	for _, k := range kind {
		if ok, err := projectorEventCompatableWith(k, e.on); !ok {
			panic(err)
		}
		e.kinds.Set(k)
	}
}

func (i ProjectorEventKind) MarshalText() ([]byte, error) {
	var s string
	if (i > 0) && (i < ProjectorEventKind_count) {
		s = i.String()
	} else {
		s = utils.UintToString(i)
	}
	return []byte(s), nil
}
