/*
* Copyright (c) 2021-present unTill Pro, Ltd.
* @author Maxim Geraskin
*
 */

package istructs

import "github.com/voedger/voedger/pkg/appdef"

type IRawEventBuilder interface {

	// ****** ArgumentObject-related builders

	// For sys.CUD command it is not called
	ArgumentObjectBuilder() IObjectBuilder
	ArgumentUnloggedObjectBuilder() IObjectBuilder

	CUDBuilder() ICUD

	// Must be last call to IRawEventBuilder
	// If err is not nil IRawEvent contains event with error
	BuildRawEvent() (raw IRawEvent, buildError error)
}

type ICUD interface {
	// Container argument can be empty for root records (documents)
	Create(qName appdef.QName) IRowWriter

	// Only record's ID and QName will be kept in the resulting event
	// It is possible to submit NullRecord (when record not found)
	Update(record IRecord) IRowWriter
}

type IObjectBuilder interface {
	IRowWriter

	// Fill object from JSON
	FillFromJSON(map[string]any)

	// Build child for nested container
	ChildBuilder(containerName string) IObjectBuilder

	// Function validates object structure
	Build() (object IObject, err error)
}

type IAbstractEvent interface {

	// If event contains error QName is consts.QNameForError
	// Otherwise is taken from params
	QName() appdef.QName

	ArgumentObject() IObject

	CUDs(func(ICUDRow) bool)

	RegisteredAt() UnixMilli
	Synced() bool

	// Valid only if Synced() true

	DeviceID() ConnectedDeviceID
	SyncedAt() UnixMilli
}

type ICUDRow interface {
	IRowReader
	IsNew() bool
	QName() appdef.QName
	ID() RecordID
	// Iterate modified fields.
	//
	// The fields are iterated in the order they were declared when the type was defined.
	//
	// #2785 - If a string- or bytes- field is emptied, then an empty string (empty byte array) will be passed to the callback iterator
	ModifiedFields(func(appdef.FieldName, interface{}) bool)
}

type IIDGenerator interface {
	NextID(rawID RecordID, t appdef.IType) (storageID RecordID, err error)
	UpdateOnSync(syncID RecordID, t appdef.IType)
}

type IRawEvent interface {
	IAbstractEvent
	ArgumentUnloggedObject() IObject

	// Context

	HandlingPartition() PartitionID
	PLogOffset() Offset
	Workspace() WSID
	WLogOffset() Offset
}

// What is kept in database
type IDbEvent interface {
	IAbstractEvent

	// Returns the event in the form of bytes that were written to storage
	// when the event was saved, or read from storage when the event was loaded
	Bytes() []byte

	// Error that occurred during event building or
	// error that describe that event is corrupted.
	Error() IEventError
}

type IEventError interface {
	// Error string or empty string if ValidEvent() == true.
	//
	// sys.Corrupted event contains error message "corrupted data".
	ErrStr() string

	// Original QName from params.
	//
	// Potentially can be invalid QName representation.
	QNameFromParams() appdef.QName

	// Returns is the event valid.
	//
	// sys.Corrupted event always invalid.
	ValidEvent() bool

	// Original bytes the event was deserialized from.
	//
	// nil if ValidEvent() == true.
	//
	// Function with unlogged params can have ValidEvent() == false and EventBytes() == nil.
	//
	// sys.Corrupted event contains bytes of corrupted event.
	OriginalEventBytes() []byte
}

// What is kept in database
type IPLogEvent interface {
	IDbEvent
	Workspace() WSID
	WLogOffset() Offset
	Release()
}

type IWLogEvent interface {
	IDbEvent
	Release()
}

type IObject interface {
	IRowReader

	QName() appdef.QName

	// Returns iterator for children in given containers
	//
	// if no containers specified then iterate all children
	Children(container ...string) func(func(IObject) bool)

	// First level qname-s
	Containers(func(string) bool)

	// Does NOT panic if it is not actually IRecord
	// Just a wrapper which uses consts.SystemField*
	// If element does not have some IRecord-related field, panic occurs when the field is read
	AsRecord() IRecord
}

// It's desirable but not necessary to call event.Release() after event using
type PLogEventsReaderCallback func(plogOffset Offset, event IPLogEvent) (err error)

// It's desirable but not necessary to call event.Release() after event using
type WLogEventsReaderCallback func(wlogOffset Offset, event IWLogEvent) (err error)

type GenericRawEventBuilderParams struct {

	// Bytes from which events are built
	// If error happens these bytes are stored and returned as part of the IDbEvent.Error() result
	EventBytes []byte

	// Context

	HandlingPartition PartitionID
	PLogOffset        Offset
	Workspace         WSID
	WLogOffset        Offset

	QName appdef.QName

	// Payload

	RegisteredAt UnixMilli
}

type SyncRawEventBuilderParams struct {
	GenericRawEventBuilderParams
	Device   ConnectedDeviceID
	SyncedAt UnixMilli
}

type NewRawEventBuilderParams struct {
	GenericRawEventBuilderParams
}
