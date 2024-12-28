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
	"github.com/voedger/voedger/pkg/appdef/internal/types"
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
	p := &Projector{
		Extension: MakeExtension(ws, name, appdef.TypeKind_Projector),
	}
	p.events = NewProjectorEvents(p)
	types.Propagate(p)
	return p
}

func (p Projector) Events() iter.Seq[appdef.IProjectorEvent] {
	return func(yield func(appdef.IProjectorEvent) bool) {
		for _, e := range p.events.events {
			if !yield(e) {
				return
			}
		}
	}
}

func (p Projector) Sync() bool { return p.sync }

func (p Projector) Triggers(op appdef.OperationKind, t appdef.IType) bool {
	for e := range p.Events() {
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
func (p *Projector) Validate() error {
	return errors.Join(
		p.Extension.Validate(),
		p.events.Validate())
}

func (p Projector) WantErrors() bool { return p.sysErrors }

func (p *Projector) setSync(sync bool) { p.sync = sync }

func (p *Projector) setWantErrors() { p.sysErrors = true }

// # Supports:
//   - appdef.IProjectorBuilder
type ProjectorBuilder struct {
	ExtensionBuilder
	p *Projector
}

func NewProjectorBuilder(p *Projector) *ProjectorBuilder {
	return &ProjectorBuilder{
		ExtensionBuilder: MakeExtensionBuilder(&p.Extension),
		p:                p,
	}
}

func (pb *ProjectorBuilder) Events() appdef.IProjectorEventsBuilder {
	return pb.p.events
}

func (pb *ProjectorBuilder) SetSync(sync bool) appdef.IProjectorBuilder {
	pb.p.setSync(sync)
	return pb
}

func (pb *ProjectorBuilder) SetWantErrors() appdef.IProjectorBuilder {
	pb.p.setWantErrors()
	return pb
}

// # Supports:
//   - appdef.IProjectorEventsBuilder
type ProjectorEvents struct {
	p      *Projector
	events []*ProjectorEvent
}

func NewProjectorEvents(p *Projector) *ProjectorEvents {
	return &ProjectorEvents{
		p:      p,
		events: make([]*ProjectorEvent, 0),
	}
}

func (ee *ProjectorEvents) Add(ops []appdef.OperationKind, flt appdef.IFilter, comment ...string) appdef.IProjectorEventsBuilder {
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
	ee.events = append(ee.events, e)
	return ee
}

// Validates projector events.
func (ee ProjectorEvents) Validate() (err error) {
	for _, e := range ee.events {
		err = errors.Join(err, e.Validate(ee.p))
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
