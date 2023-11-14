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
	Events(func(IProjectorEvent))

	// Returns projector states.
	//
	// State is a storage to get data.
	States() QNames

	// Returns projector intents.
	//
	// Intent is a storage to put data.
	Intents() QNames
}

// Describe event to trigger the projector.
type IProjectorEvent interface {
	IComment

	// Returns set of event kind to trigger.
	Kind() []ProjectorEventKind

	// Returns record type to trigger projector.
	On() IRecord
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
	// Record can be some record type or QNameANY.
	//
	// If event kind is missed then default is ProjectorEventKind_Any.
	//
	// # Panics:
	//	- if record is empty (NullQName) or unknown record type.
	AddEvent(record QName, event ...ProjectorEventKind) IProjectorBuilder

	// Sets event comment.
	//
	// # Panics:
	//	- if event for record is not added
	SetEventComment(record QName, comment ...string) IProjectorBuilder

	// Adds state to the projector.
	AddState(...QName) IProjectorBuilder

	// Adds intent to the projector.
	AddIntent(...QName) IProjectorBuilder
}
