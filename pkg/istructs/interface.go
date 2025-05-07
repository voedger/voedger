/*
* Copyright (c) 2021-present unTill Pro, Ltd.
* @author Maxim Geraskin
*
 */

package istructs

import (
	"context"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
)

// Structs can be changed on-the-fly, so AppStructs() are taken for each message (request) to be handled
type IAppStructsProvider interface {
	// Returns AppStructs for builtin application with given name.
	//
	// ErrAppNotFound can be returned.
	//
	// @ConcurrentAccess
	BuiltIn(appdef.AppQName) (IAppStructs, error)

	// // TODO
	// Sidecar(appdef.AppQName) (IAppStructs, error)

	// AppStructs(appdef.AppQName) (IAppStructs, error)

	// Creates a new AppStructs for user application with given name, id and definition.
	//
	// @ConcurrentAccess
	New(appdef.AppQName, appdef.IAppDef, ClusterAppID, NumAppWorkspaces) (IAppStructs, error)
}

type IAppStructs interface {

	// ************************************************************
	// Dynamic data, kind of global variables

	Events() IEvents

	Records() IRecords

	ViewRecords() IViewRecords

	// ************************************************************
	// Runtime helpers

	ObjectBuilder(appdef.QName) IObjectBuilder

	// ************************************************************
	// Static data, kind of constants

	// Working with resources like functions, images (in the future)
	// Function can be inside WASM, container, executable, jar, zip etc.
	Resources() IResources

	// ************************************************************
	// Application definition, kind of RTTI, reflection

	// AppDef
	AppDef() appdef.IAppDef

	ClusterAppID() ClusterAppID
	AppQName() appdef.AppQName

	IsFunctionRateLimitsExceeded(funcQName appdef.QName, wsid WSID) bool // FIXME: eliminate, use the one from iappparts?
	// Describe package names
	DescribePackageNames() []string

	// Describe package content
	DescribePackage(pkgName string) interface{}

	SyncProjectors() Projectors
	AsyncProjectors() Projectors

	CUDValidators() []CUDValidator
	EventValidators() []EventValidator

	NumAppWorkspaces() NumAppWorkspaces

	AppTokens() IAppTokens // FIXME: check what is for?

	// to execute PutWLog and ApplyRecords skipping SequencesTrustLevel on partition recoevery
	// panics if IPLogEvent was not loaded from the database
	GetEventReapplier(IPLogEvent) IEventReapplier

	SeqTypes() map[QNameID]map[QNameID]uint64

	QNameID(qName appdef.QName) (QNameID, error)
}

// need to re-apply an already stored PLog
// does not consider SequencesTrustLevel

type IEventReapplier interface {
	PutWLog() error
	ApplyRecords() error
}

type IEvents interface {
	GetSyncRawEventBuilder(params SyncRawEventBuilderParams) IRawEventBuilder
	GetNewRawEventBuilder(params NewRawEventBuilderParams) IRawEventBuilder

	// BuildPLogEvent builds PLogEvent from IRawEvent, but do not puts result PLogEvent into PLog.
	//
	// Should be used to obtain `sys.Corrupted` events only.
	//
	// # Panics
	//	- if raw event is not `sys.Corrupted`
	//	- if raw event PLogOffset is not null
	BuildPLogEvent(IRawEvent) IPLogEvent

	// @ConcurrentAccess RW
	// buildOrValidationErr taken either BuildRawEvent() or from extra validation
	//
	// Raw event `ev` valid until `event.Release()`
	PutPlog(ev IRawEvent, buildOrValidationErr error, generator IIDGenerator) (event IPLogEvent, saveErr error)

	// @ConcurrentAccess RW
	PutWlog(IPLogEvent) error

	// @ConcurrentAccess R
	// consts.ReadToTheEnd can be used for the toReadCount parameter
	ReadPLog(ctx context.Context, partition PartitionID, offset Offset, toReadCount int, cb PLogEventsReaderCallback) (err error)
	ReadWLog(ctx context.Context, workspace WSID, offset Offset, toReadCount int, cb WLogEventsReaderCallback) (err error)
}

type IRecords interface {
	// Apply all CUDs, ODocs and WDocs from the given IPLogEvent
	// @ConcurrentAccess RW
	// Panics if event is not valid
	Apply(event IPLogEvent) (err error)

	// cb gets new version of each record affected by CUDs
	// Panics if event is not valid
	Apply2(event IPLogEvent, cb func(r IRecord)) (err error)

	// @ConcurrentAccess RW
	//
	// Record system fields sys.QName and sys.ID should be filled.
	// Data type name in sys.QName should be storable record type.
	// Record ID in sys.ID should not be a raw ID.
	//
	// Attention! This method does not perform a full validation of the recorded data :
	// - The values of referenced record IDs are not checked
	// - The fullness of the required fields is not checked
	PutJSON(WSID, map[appdef.FieldName]any) error

	// @ConcurrentAccess R
	// Can read GDoc, CDoc, ODoc, WDoc records
	// If record not found NullRecord with QName() == NullQName is returned
	// NullRecord.WSID & ID will be taken from arguments
	Get(workspace WSID, highConsistency bool, id RecordID) (record IRecord, err error)

	GetBatch(workspace WSID, highConsistency bool, ids []RecordGetBatchItem) (err error)

	// @ConcurrentAccess R
	// qName must be a singleton
	// If record not found NullRecord with QName() == NullQName is returned
	GetSingleton(workspace WSID, qName appdef.QName) (record IRecord, err error)

	GetSingletonID(qName appdef.QName) (RecordID, error)
}

type RecordGetBatchItem struct {
	ID     RecordID // in
	Record IRecord  // out
}

type IViewRecords interface {

	// Builders panic if QName not found

	KeyBuilder(view appdef.QName) IKeyBuilder
	NewValueBuilder(view appdef.QName) IValueBuilder
	UpdateValueBuilder(view appdef.QName, existing IValue) IValueBuilder

	// All key fields must be specified (panic)
	// Key & value must be from the same QName (panic)
	Put(workspace WSID, key IKeyBuilder, value IValueBuilder) (err error)

	PutBatch(workspace WSID, batch []ViewKV) (err error)

	// @ConcurrentAccess RW
	//
	// View name should be passed in sys.QName field.
	// All key fields should be filled.
	//
	// Attention! This method does not perform a full validation of the recorded data :
	// - The values of referenced record IDs are not checked
	// - The fullness of the required view value fields is not checked
	PutJSON(WSID, map[appdef.FieldName]any) error

	// All fields must be filled in in the key (panic otherwise)
	Get(workspace WSID, key IKeyBuilder) (value IValue, err error)

	GetBatch(workspace WSID, kv []ViewRecordGetBatchItem) (err error)

	// All fields of key.PartitionKey MUST be specified (panic)
	// Zero or more fields of key.ClusteringColumns can be specified
	// If last clustering column has variable length it can be filled partially
	Read(ctx context.Context, workspace WSID, key IKeyBuilder, cb ValuesCallback) (err error)
}

type ViewRecordGetBatchItem struct {
	Key   IKeyBuilder // in
	Ok    bool        // out
	Value IValue      // out
}

type ViewKV struct {
	Key   IKeyBuilder
	Value IValueBuilder
}

type ValuesCallback func(key IKey, value IValue) (err error)

type IResources interface {

	// If resource not found then {ResourceKind_null, QNameForNullResource) is returned
	// Currently resources are ICommandFunction and IQueryFunction
	QueryResource(resource appdef.QName) (r IResource)

	// Enumerates all application resources
	Resources(func(appdef.QName) bool)
}

// Same as itokens.ITokens but works for App specified in IAppTokensFactory
// App is configured per interface instance
// placed here because otherwise IAppStructs.AppTokens() would depend on itokens-payloads
type IAppTokens interface {
	// Calls istructs.IssueToken for given App
	IssueToken(duration time.Duration, pointerToPayload interface{}) (token string, err error)
	// ErrTokenIssuedForAnotherApp is returned (check using errors.Is(...)) when token is issued for another application
	ValidateToken(token string, pointerToPayload interface{}) (gp GenericPayload, err error)
}

// All payloads must inherit this payload
type GenericPayload struct {
	AppQName appdef.AppQName
	Duration time.Duration
	IssuedAt time.Time
}
