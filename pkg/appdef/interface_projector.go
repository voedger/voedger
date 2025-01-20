/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Projector is a extension that executes every time when some event is triggered and data need to be updated.
type IProjector interface {
	IExtension

	// Returns is synchronous projector.
	Sync() bool

	// Returns events that triggers this projector.
	Events() []IProjectorEvent

	// Returns whether this projector triggers with the specified operation and type
	Triggers(OperationKind, IType) bool

	// Returns is projector is able to handle `sys.Error` events.
	// False by default.
	WantErrors() bool
}

type IProjectorEvent interface {
	IWithComments

	// Returns is triggered by specified operation.
	Op(OperationKind) bool

	// Returns triggered operations.
	Ops() []OperationKind

	// Returns filter of triggered types.
	Filter() IFilter
}

type IProjectorBuilder interface {
	IExtensionBuilder

	// Returns events builder
	Events() IProjectorEventsBuilder

	// Sets is synchronous projector.
	SetSync(bool) IProjectorBuilder

	// Sets is projector is able to handle `sys.Error` events.
	SetWantErrors() IProjectorBuilder
}

type IProjectorEventsBuilder interface {
	// Adds new event.
	//
	// # Panics:
	//	 - if specified operations are incompatible,
	//	 - if matched objects can not to be used with specified operations.
	Add(ops []OperationKind, flt IFilter, comment ...string) IProjectorEventsBuilder
}

type IProjectorsBuilder interface {
	// Adds new projector.
	//
	// # Panics:
	//   - if name is empty or invalid,
	//   - if type with the same name already exists.
	AddProjector(name QName) IProjectorBuilder
}
