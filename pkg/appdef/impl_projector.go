/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"iter"

	"github.com/voedger/voedger/pkg/goutils/set"
)

// # Supports:
//   - IProjector
type projector struct {
	extension
	sync      bool
	sysErrors bool
	events    *projectorEvents
}

func newProjector(app *appDef, ws *workspace, name QName) *projector {
	prj := &projector{
		extension: makeExtension(app, ws, name, TypeKind_Projector),
	}
	prj.events = newProjectorEvents(prj)
	ws.appendType(prj)
	return prj
}

func (prj projector) Events() iter.Seq[IProjectorEvent] {
	return func(yield func(IProjectorEvent) bool) {
		for _, e := range prj.events.events {
			if !yield(e) {
				return
			}
		}
	}
}

func (prj projector) Sync() bool { return prj.sync }

func (prj projector) Triggers(op OperationKind, t IType) bool {
	for e := range prj.Events() {
		if e.Op(op) {
			if e.Filter().Match(t) {
				return true
			}
		}
	}
	return false
}

// Validates projector.
//
// # Error if:
//   - some event filter has no matches in the workspace
//   - some event filtered type can not trigger projector
func (prj *projector) Validate() error {
	return errors.Join(
		prj.extension.Validate(),
		prj.events.validate())
}

func (prj projector) WantErrors() bool { return prj.sysErrors }

func (prj *projector) setSync(sync bool) { prj.sync = sync }

func (prj *projector) setWantErrors() { prj.sysErrors = true }

// # Supports:
//   - IProjectorBuilder
type projectorBuilder struct {
	extensionBuilder
	*projector
}

func newProjectorBuilder(projector *projector) *projectorBuilder {
	return &projectorBuilder{
		extensionBuilder: makeExtensionBuilder(&projector.extension),
		projector:        projector,
	}
}

func (pb *projectorBuilder) Events() IProjectorEventsBuilder {
	return pb.projector.events
}

func (pb *projectorBuilder) SetSync(sync bool) IProjectorBuilder {
	pb.projector.setSync(sync)
	return pb
}

func (pb *projectorBuilder) SetWantErrors() IProjectorBuilder {
	pb.projector.setWantErrors()
	return pb
}

// # Supports:
//   - IProjectorEventsBuilder
type projectorEvents struct {
	prj    *projector
	events []*projectorEvent
}

func newProjectorEvents(prj *projector) *projectorEvents {
	return &projectorEvents{
		prj:    prj,
		events: make([]*projectorEvent, 0),
	}
}

func (ev *projectorEvents) Add(ops []OperationKind, flt IFilter, comment ...string) IProjectorEventsBuilder {
	if !ProjectorOperations.ContainsAll(ops...) {
		panic(ErrUnsupported("projector operations %v", ops))
	}
	if flt == nil {
		panic(ErrMissed("filter"))
	}
	e := &projectorEvent{
		ops: set.From(ops...),
		flt: flt,
	}
	e.comment.setComment(comment...)
	ev.events = append(ev.events, e)
	return ev
}

// Validates projector events.
func (ev projectorEvents) validate() (err error) {
	for _, e := range ev.events {
		err = errors.Join(err, e.validate(ev.prj))
	}
	return err
}

// # Supports:
//   - IProjectorEvent
type projectorEvent struct {
	comment
	ops set.Set[OperationKind]
	flt IFilter
}

func (e projectorEvent) Filter() IFilter { return e.flt }

func (e projectorEvent) Op(o OperationKind) bool { return e.ops.Contains(o) }

func (e projectorEvent) Ops() iter.Seq[OperationKind] { return e.ops.Values() }

// Validates projector event.
func (ev projectorEvent) validate(prj IProjector) (err error) {
	cnt := 0
	for t := range prj.Workspace().Types() {
		if TypeKind_ProjectorTriggers.Contains(t.Kind()) {
			if ev.flt.Match(t) {
				cnt++
			}
		}
	}
	if cnt == 0 {
		err = errors.Join(err, ErrFilterHasNoMatches(prj, ev.flt, prj.Workspace()))
	}
	return err
}
