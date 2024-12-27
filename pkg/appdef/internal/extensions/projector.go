/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions

import (
	"errors"
	"iter"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
	"github.com/voedger/voedger/pkg/goutils/set"
)

// # Supports:
//   - appdef.IProjector
type Projector struct {
	Extension
	sync      bool
	sysErrors bool
	events    *ProjectorEvents
}

func NewProjector(ws appdef.IWorkspace, name appdef.QName) *Projector {
	prj := &Projector{
		Extension: MakeExtension(ws, name, appdef.TypeKind_Projector),
	}
	prj.events = NewProjectorEvents(prj)
	return prj
}

func (prj Projector) Events() iter.Seq[appdef.IProjectorEvent] {
	return func(yield func(appdef.IProjectorEvent) bool) {
		for _, e := range prj.events.events {
			if !yield(e) {
				return
			}
		}
	}
}

func (prj Projector) Sync() bool { return prj.sync }

func (prj Projector) Triggers(op appdef.OperationKind, t appdef.IType) bool {
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
func (prj *Projector) Validate() error {
	return errors.Join(
		prj.Extension.Validate(),
		prj.events.Validate())
}

func (prj Projector) WantErrors() bool { return prj.sysErrors }

func (prj *Projector) setSync(sync bool) { prj.sync = sync }

func (prj *Projector) setWantErrors() { prj.sysErrors = true }

// # Supports:
//   - appdef.IProjectorBuilder
type ProjectorBuilder struct {
	ExtensionBuilder
	*Projector
}

func NewProjectorBuilder(projector *Projector) *ProjectorBuilder {
	return &ProjectorBuilder{
		ExtensionBuilder: MakeExtensionBuilder(&projector.Extension),
		Projector:        projector,
	}
}

func (pb *ProjectorBuilder) Events() appdef.IProjectorEventsBuilder {
	return pb.Projector.events
}

func (pb *ProjectorBuilder) SetSync(sync bool) appdef.IProjectorBuilder {
	pb.Projector.setSync(sync)
	return pb
}

func (pb *ProjectorBuilder) SetWantErrors() appdef.IProjectorBuilder {
	pb.Projector.setWantErrors()
	return pb
}

// # Supports:
//   - appdef.IProjectorEventsBuilder
type ProjectorEvents struct {
	prj    *Projector
	events []*ProjectorEvent
}

func NewProjectorEvents(prj *Projector) *ProjectorEvents {
	return &ProjectorEvents{
		prj:    prj,
		events: make([]*ProjectorEvent, 0),
	}
}

func (ev *ProjectorEvents) Add(ops []appdef.OperationKind, flt appdef.IFilter, comment ...string) appdef.IProjectorEventsBuilder {
	if !appdef.ProjectorOperations.ContainsAll(ops...) {
		panic(appdef.ErrUnsupported("projector operations %v", ops))
	}
	if flt == nil {
		panic(appdef.ErrMissed("filter"))
	}
	e := &ProjectorEvent{
		ops: set.From(ops...),
		flt: flt,
	}
	comments.SetComment(&e.WithComments, comment...)
	ev.events = append(ev.events, e)
	return ev
}

// Validates projector events.
func (ev ProjectorEvents) Validate() (err error) {
	for _, e := range ev.events {
		err = errors.Join(err, e.Validate(ev.prj))
	}
	return err
}

// # Supports:
//   - appdef.IProjectorEvent
type ProjectorEvent struct {
	comments.WithComments
	ops set.Set[appdef.OperationKind]
	flt appdef.IFilter
}

func (e ProjectorEvent) Filter() appdef.IFilter { return e.flt }

func (e ProjectorEvent) Op(o appdef.OperationKind) bool { return e.ops.Contains(o) }

func (e ProjectorEvent) Ops() iter.Seq[appdef.OperationKind] { return e.ops.Values() }

// Validates projector event.
func (e ProjectorEvent) Validate(prj appdef.IProjector) (err error) {
	cnt := 0
	for t := range prj.Workspace().Types() {
		if appdef.TypeKind_ProjectorTriggers.Contains(t.Kind()) {
			if e.flt.Match(t) {
				cnt++
				break // is enough
			}
		}
	}
	if cnt == 0 {
		err = errors.Join(err, appdef.ErrFilterHasNoMatches(prj, e.flt, prj.Workspace()))
	}
	return err
}
