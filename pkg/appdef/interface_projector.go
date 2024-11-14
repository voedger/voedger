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

	// Events to trigger.
	Events() IProjectorEvents

	// Returns is projector is able to handle `sys.Error` events.
	// False by default.
	WantErrors() bool
}

// Describe all events to trigger the projector.
type IProjectorEvents interface {
	// Enumerate events to trigger the projector.
	//
	// Events enumerated in alphabetical QNames order.
	Enum(func(IProjectorEvent))

	// Returns event by name.
	//
	// Returns nil if event not found.
	Event(QName) IProjectorEvent

	// Returns number of events.
	Len() int

	// Returns events to trigger as map.
	Map() map[QName][]ProjectorEventKind
}

// Describe event to trigger the projector.
type IProjectorEvent interface {
	IWithComments

	// Returns type to trigger projector.
	//
	// This can be a record or command.
	On() IType

	// Returns set (sorted slice) of event kind to trigger.
	Kind() []ProjectorEventKind
}

// Events enumeration to trigger the projector
type ProjectorEventKind uint8

//go:generate stringer -type=ProjectorEventKind -output=stringer_projectoreventkind.go

const (
	ProjectorEventKind_Insert ProjectorEventKind = iota + 1
	ProjectorEventKind_Update
	ProjectorEventKind_Activate
	ProjectorEventKind_Deactivate
	ProjectorEventKind_Execute
	ProjectorEventKind_ExecuteWithParam

	ProjectorEventKind_count
)

// ProjectorEventKind_AnyChanges describes events for record any change.
var ProjectorEventKind_AnyChanges = []ProjectorEventKind{
	ProjectorEventKind_Insert,
	ProjectorEventKind_Update,
	ProjectorEventKind_Activate,
	ProjectorEventKind_Deactivate,
}

type IProjectorBuilder interface {
	IExtensionBuilder

	// Sets is synchronous projector.
	SetSync(bool) IProjectorBuilder

	// Events builder.
	Events() IProjectorEventsBuilder

	// Sets is projector is able to handle `sys.Error` events.
	SetWantErrors() IProjectorBuilder
}

type IProjectorEventsBuilder interface {
	// Adds event to trigger the projector.
	//
	// QName can be some record type or command. QName can be one of QNameAny××× compatible substitutions.
	//
	// If event kind is missed then default is:
	//   - ProjectorEventKind_Any for GDoc/GRecords, CDoc/CRecords and WDoc/WRecords
	//	 - ProjectorEventKind_Execute for Commands
	//	 - ProjectorEventKind_ExecuteWith for Objects and ODocs
	//
	// # Panics:
	//	- if QName is empty (NullQName)
	//	- if QName type is not a record and not a command
	//	- if event kind is not applicable for QName type.
	Add(on QName, event ...ProjectorEventKind) IProjectorEventsBuilder

	// Sets event comment.
	//
	// # Panics:
	//	- if event for QName is not added.
	SetComment(on QName, comment ...string) IProjectorEventsBuilder
}

type IProjectorsBuilder interface {
	// Adds new projector.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddProjector(QName) IProjectorBuilder
}
