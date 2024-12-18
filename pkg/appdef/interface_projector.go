/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "iter"

// Projector is a extension that executes every time when some event is triggered and data need to be updated.
type IProjector interface {
	IExtension

	// Returns is synchronous projector.
	Sync() bool

	// Returns is this projector triggered by specified operation.
	Op(OperationKind) bool

	// Returns operations that triggers this projector.
	Ops() iter.Seq[OperationKind]

	// Returns filter of types on which projector is applicable.
	Filter() IFilter

	// Returns is projector is able to handle `sys.Error` events.
	// False by default.
	WantErrors() bool
}

type IProjectorBuilder interface {
	IExtensionBuilder

	// Sets is synchronous projector.
	SetSync(bool) IProjectorBuilder

	// Sets is projector is able to handle `sys.Error` events.
	SetWantErrors() IProjectorBuilder
}

type IProjectorsBuilder interface {
	// Adds new projector.
	//
	// # Panics:
	//   - if name is empty or invalid,
	//   - if type with the same name already exists,
	//	 - if specified operations are incompatible,
	//	 - if matched objects can not to be used with specified operations.
	AddProjector(name QName, ops []OperationKind, flt IFilter, comment ...string) IProjectorBuilder
}
