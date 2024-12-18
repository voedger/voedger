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
	ops       set.Set[OperationKind]
	flt       IFilter
}

func newProjector(app *appDef, ws *workspace, name QName, ops []OperationKind, flt IFilter, comment ...string) *projector {
	if !ProjectorOperations.ContainsAll(ops...) {
		panic(ErrUnsupported("projector operations %v", ops))
	}

	opSet := set.From(ops...)
	if compatible, err := isCompatibleOperations(opSet); !compatible {
		panic(err)
	}
	if flt == nil {
		panic(ErrMissed("filter"))
	}
	prj := &projector{
		extension: makeExtension(app, ws, name, TypeKind_Projector),
		ops:       opSet,
		flt:       flt,
	}
	for t := range FilterMatches(prj.Filter(), ws.Types()) {
		if err := prj.validateOnType(t); err != nil {
			panic(err)
		}
	}
	prj.typ.comment.setComment(comment...)
	ws.appendType(prj)
	return prj
}

func (prj projector) Filter() IFilter { return prj.flt }

func (prj projector) Op(o OperationKind) bool { return prj.ops.Contains(o) }

func (prj projector) Ops() iter.Seq[OperationKind] { return prj.ops.Values() }

func (prj projector) Sync() bool { return prj.sync }

// Validates projector.
//
// # Error if:
//   - filter has no matches in the workspace
//   - some filtered type can not trigger projector. See validateOnType
func (prj projector) Validate() (err error) {
	err = prj.extension.Validate()

	cnt := 0
	for t := range FilterMatches(prj.Filter(), prj.Workspace().Types()) {
		err = errors.Join(err, prj.validateOnType(t))
		cnt++
	}

	if cnt == 0 {
		err = errors.Join(err, ErrFilterHasNoMatches(prj, prj.Filter(), prj.Workspace()))
	}

	return err
}

func (prj projector) WantErrors() bool { return prj.sysErrors }

func (prj *projector) setSync(sync bool) { prj.sync = sync }

func (prj *projector) setWantErrors() { prj.sysErrors = true }

func (prj projector) validateOnType(t IType) error {
	if !TypeKind_ProjectorTriggers.Contains(t.Kind()) {
		return ErrUnsupported("%v can not trigger projector", t)
	}
	return nil
}

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

func (pb *projectorBuilder) SetSync(sync bool) IProjectorBuilder {
	pb.projector.setSync(sync)
	return pb
}

func (pb *projectorBuilder) SetWantErrors() IProjectorBuilder {
	pb.projector.setWantErrors()
	return pb
}
