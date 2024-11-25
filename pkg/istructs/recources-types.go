/*
* Copyright (c) 2021-present unTill Pro, Ltd.
* @author Maxim Geraskin
*
 */

package istructs

import (
	"context"
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
)

//go:generate stringer -type=ResourceKindType
type ResourceKindType uint8

const (
	ResourceKind_null ResourceKindType = iota
	ResourceKind_CommandFunction
	ResourceKind_QueryFunction
	ResourceKind_FakeLast
)

type IResource interface {
	// Ref. ResourceKind_* constants
	Kind() ResourceKindType
	QName() appdef.QName
}

// ******************* Functions **************************

type IFunction interface {
	IResource
}

type ICommandFunction interface {
	IFunction
	Exec(args ExecCommandArgs) error
}

type IQueryFunction interface {
	IFunction
	// panics if created by not NewQueryFunctionCustomResult(). Actually needed for q.sys.Collection only
	ResultType(args PrepareArgs) appdef.QName
	Exec(ctx context.Context, args ExecQueryArgs, callback ExecQueryCallback) error
}

type PrepareArgs struct {
	Workpiece      interface{}
	ArgumentObject IObject
	WSID           WSID
	Workspace      appdef.IWorkspace
}

type CommandPrepareArgs struct {
	PrepareArgs
	ArgumentUnloggedObject IObject
}

type ExecCommandArgs struct {
	CommandPrepareArgs
	State   IState
	Intents IIntents
}

type ExecQueryCallback func(object IObject) error

type ExecQueryArgs struct {
	PrepareArgs
	State   IState
	Intents IIntents
}

type IState interface {
	// NewKey returns a Key builder for specified storage and entity name
	KeyBuilder(storage, entity appdef.QName) (builder IStateKeyBuilder, err error)

	CanExist(key IStateKeyBuilder) (value IStateValue, ok bool, err error)

	CanExistAll(keys []IStateKeyBuilder, callback StateValueCallback) (err error)

	MustExist(key IStateKeyBuilder) (value IStateValue, err error)

	MustExistAll(keys []IStateKeyBuilder, callback StateValueCallback) (err error)

	MustNotExist(key IStateKeyBuilder) (err error)

	MustNotExistAll(keys []IStateKeyBuilder) (err error)

	// Read reads all values according to the get and return them in callback
	Read(key IStateKeyBuilder, callback ValueCallback) (err error)

	// For projectors
	PLogEvent() IPLogEvent

	// For commands
	CommandPrepareArgs() CommandPrepareArgs

	// For queries
	QueryPrepareArgs() PrepareArgs
	QueryCallback() ExecQueryCallback

	App() appdef.AppQName
	AppStructs() IAppStructs
}

type IIntents interface {
	// NewValue returns a new value builder for given get
	// If a value with the same get already exists in storage, it will be replaced
	NewValue(key IStateKeyBuilder) (builder IStateValueBuilder, err error)

	// UpdateValue returns a value builder to update existing value
	UpdateValue(key IStateKeyBuilder, existingValue IStateValue) (builder IStateValueBuilder, err error)

	// returns nil when not found
	FindIntent(key IStateKeyBuilder) IStateValueBuilder

	FindIntentWithOpKind(key IStateKeyBuilder) (IStateValueBuilder, bool)

	IntentsCount() int
	// iterate over all intents
	Intents(iterFunc func(key IStateKeyBuilder, value IStateValueBuilder, isNew bool))
}
type IPkgNameResolver interface {
	// Returns package path by package local name.
	//
	// Returns empty string if not found
	PackageFullPath(localName string) string

	// Returns package local name by package path.
	//
	// Returns empty string if not found
	PackageLocalName(fullPath string) string
}
type IStateValue interface {
	IRowReader
	AsValue(name string) IStateValue
	Length() int
	GetAsString(index int) string
	GetAsBytes(index int) []byte
	GetAsInt32(index int) int32
	GetAsInt64(index int) int64
	GetAsFloat32(index int) float32
	GetAsFloat64(index int) float64
	GetAsQName(index int) appdef.QName
	GetAsBool(index int) bool
	GetAsValue(index int) IStateValue
}
type IStateRecordValue interface {
	AsRecord() IRecord
}

type IStateViewValue interface {
	AsRecord(name string) IRecord
}

type IStateWLogValue interface {
	AsEvent() IWLogEvent
}

type IStateValueBuilder interface {
	IRowWriter

	BuildValue() IStateValue // Currently used in testState and for the intents in the bundled storage. Must return nil of not supported by storage.

	Equal(to IStateValueBuilder) bool // used in testState
}
type IStateViewValueBuilder interface {
	PutRecord(name string, record IRecord)
}
type IStateKeyBuilder interface {
	IKeyBuilder
	fmt.Stringer
	Storage() appdef.QName
	Entity() appdef.QName
}
type StateValueCallback func(key IKeyBuilder, value IStateValue, ok bool) (err error)
type ValueCallback func(key IKey, value IStateValue) (err error)

//go:generate stringer -type=RateLimitKind
type RateLimitKind uint8

type RateLimit struct {
	Period                time.Duration
	MaxAllowedPerDuration uint32
}
