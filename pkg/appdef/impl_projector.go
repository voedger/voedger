/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/robfig/cron/v3"
)

// # Implements:
//   - IProjector
type projector struct {
	extension
	sync         bool
	sysErrors    bool
	events       *events
	cronSchedule string
}

func newProjector(app *appDef, name QName) *projector {
	prj := &projector{
		extension: makeExtension(app, name, TypeKind_Projector),
		events:    newProjectorEvents(app),
	}
	app.appendType(prj)
	return prj
}

func (prj projector) CronSchedule() string { return prj.cronSchedule }

func (prj projector) Events() IProjectorEvents { return prj.events }

func (prj projector) Sync() bool { return prj.sync }

func (prj projector) WantErrors() bool { return prj.sysErrors }

func (prj *projector) setCronSchedule(cs string) { prj.cronSchedule = cs }

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

func (pb *projectorBuilder) SetCronSchedule(cs string) IProjectorBuilder {
	pb.projector.setCronSchedule(cs)
	return pb
}

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

	if (len(prj.events.events) == 0) && (prj.cronSchedule == "") {
		err = errors.Join(err, ErrMissed("%v events", prj))
	}

	if prj.cronSchedule != "" {
		_, e := cron.ParseStandard(prj.cronSchedule)
		if e != nil {
			err = errors.Join(err, enrichError(e, "%v cron schedule", prj))
		}

		if prj.intents.Len() > 0 {
			err = errors.Join(err, ErrUnsupported("%v with schedule can't have intents", prj))
		}
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

	t := ee.app.TypeByName(on)
	if t == nil {
		panic(ErrTypeNotFound(on))
	}
	switch t.Kind() {
	case TypeKind_GDoc, TypeKind_GRecord, TypeKind_CDoc, TypeKind_CRecord, TypeKind_WDoc, TypeKind_WRecord, // CUD
		TypeKind_Command,               // Execute
		TypeKind_ODoc, TypeKind_Object: // Execute with
		e, ok := ee.events[on]
		if ok {
			e.addKind(event...)
		} else {
			e = newEvent(t, event...)
			ee.events[on] = e
		}
		ee.eventsMap[on] = e.Kind()
	default:
		panic(ErrIncompatible("%v is not applicable for projector event", t))
	}
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
	kinds uint64 // bitmap[ProjectorEventKind]
}

func newEvent(on IType, kind ...ProjectorEventKind) *event {
	p := &event{on: on}

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

func (e *event) Kind() (kinds []ProjectorEventKind) {
	for k := ProjectorEventKind(1); k < ProjectorEventKind_Count; k++ {
		if e.kinds&(1<<k) != 0 {
			kinds = append(kinds, k)
		}
	}
	return kinds
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
		if !k.typeCompatible(e.on.Kind()) {
			panic(ErrIncompatible("event kind «%s» is not compatible with %v", k.TrimString(), e.on))
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
