/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Projector is a type of object that executes every time when some event is triggered and data need to be updated.
type IProjector interface {
	IType

	// Returns is synchronous projector.
	Sync() bool

	// Returns extension for projector.
	Extension() IExtension

	// Enumerate events to trigger the projector.
	//
	// Events enumerated in alphabetical QNames order.
	Events(func(IProjectorEvent))

	// Returns projector states.
	//
	// State is a storage to get data.
	//
	// States storages enumerated in alphabetical QNames order.
	// Names slice in every intent storage is sorted and deduplicated.
	States(func(storage QName, names QNames))

	// Returns projector intents.
	//
	// Intent is a storage to put data.
	//
	// Intents storages enumerated in alphabetical QNames order.
	// Names slice in every intent storage is sorted and deduplicated.
	Intents(func(storage QName, names QNames))
}

// Describe event to trigger the projector.
type IProjectorEvent interface {
	IComment

	// Returns type to trigger projector.
	//
	// This can be a record or command.
	On() IType

	// Returns set (sorted slice) of event kind to trigger.
	Kind() []ProjectorEventKind
}

// Events enumeration to trigger the projector.
//
// Ref. to projector-event-kind.go for constants and methods
type ProjectorEventKind uint8

type IProjectorBuilder interface {
	IProjector
	ITypeBuilder

	// Sets is synchronous projector.
	SetSync(bool) IProjectorBuilder

	// Sets engine.
	//
	// If name is empty then default is projector type name (entity part only without package).
	//
	// # Panics:
	//	- if name is invalid identifier
	SetExtension(name string, engine ExtensionEngineKind, comment ...string) IProjectorBuilder

	// Adds event to trigger the projector.
	//
	// QName can be some record type or command.
	//
	// If event kind is missed then default is ProjectorEventKind_Any for records and ProjectorEventKind_Execute for commands.
	//
	// # Panics:
	//	- if QName is empty (NullQName)
	//	- if QName type is not a record and not a command
	//	- if event kind is not applicable for QName type.
	AddEvent(on QName, event ...ProjectorEventKind) IProjectorBuilder

	// Sets event comment.
	//
	// # Panics:
	//	- if event for QName is not added
	SetEventComment(on QName, comment ...string) IProjectorBuilder

	// Adds state to the projector.
	//
	// If storage with name is already exists in states then names will be added to existing storage.
	AddState(storage QName, names ...QName) IProjectorBuilder

	// Adds intent to the projector.
	//
	// If storage with name is already exists in intents then names will be added to existing storage.
	AddIntent(storage QName, names ...QName) IProjectorBuilder
}
