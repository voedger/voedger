/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package iextsse

import (
	"net/url"
)

// ************************************************************
// `SSE` stands for `State Storage Extension`.

// Shall be created once per VVM per VvmFactory type
type ISSEVvmFactory interface {

	// storageModuleURL shall be valid until the factory is released.
	NewAppFactory(storageModuleURL url.URL) (ISSEAppVerFactory, error)
}

// Use: VersionConfigPtr <load VersionConfig from json> SetConfig {NewPartitionFactory}
type ISSEAppVerFactory interface {
	// Released when there are no more partitions for the application version.
	IReleasable

	// nil means "no config".
	// ConfigPtr shall be loaded from a json config  (if the config exists) prior calling SetConfig.
	ConfigPtr() *any

	// Shall be called once prior any call to the NewPartitionFactory() and after the ConfigPtr() result is loaded.
	ApplyConfigs(cfg *SSECommonConfig) error

	// NewPartitionFactory is called once per partition per application version.
	// Existing is one of the active factories for the partition.
	// Existing is nil if this is the first factory for the partition.
	NewPartitionFactory(partitionID PartitionID, existing ISSEPartitionFactory) ISSEPartitionFactory
}

type SSECommonConfig struct {
	Logger ISSELogger
}

type ISSELogger interface {
	Error(args ...interface{})
	Warning(args ...interface{})
	Info(args ...interface{})
	Verbose(args ...interface{})
}

type ISSEPartitionFactory interface {
	// Released when the partition factory is not the current factory and there are no more active StateStorage-s for the partition factory.
	IReleasable

	// Shall be called when a new state storage extension instance is needed (e.g. for every command/query processing)
	NewStateStorage(WSID uint64) ISSEStateStorage
}

// ************************************************************
// ISSE* interfaces

// Will be type-asserted to ISSEStateStorageWith* interfaces.
type ISSEStateStorage interface {
	IReleasable
}

type ISSEStateStorageWithGet interface {
	// Must return within 4 seconds.
	Get(key ISSEKey) (v ISSEValue, ok bool, err error)
}

type ISSEStateStorageWithInsert interface {
	DummyWithInsert()
}

type ISSEStateStorageWithUpdate interface {
	DummyWithUpdate()
}

type ISSEStateStorageWithApplyBatch interface {
	// Must return within 4 seconds.
	ApplyBatch(items []ISSEApplyBatchItem) (err error)
}

type ISSEApplyBatchItem struct {
	Key   ISSEKey
	Value ISSEValue
	IsNew bool
}

// ************************************************************

// As* methods panic if the requested type is not compatible with the value
// If value is missing, As* methods return a zero value and false.
type ISSEKey interface {
	Namespace() string
	Name() string
	AsInt64(name string) (value int64, ok bool)
	AsString(name string) (value string, ok bool)
}

// ISSEValue is a read-only interface.
// As* methods panics if the requested type is not compatible with the value or the `name` parameter is invalid.
// If the `name` is valid, but the value is missing, As* methods returns a zero value.
type ISSEValue interface {
	IReleasable

	// Basic types

	AsInt64(name string) int64
	AsFloat64(name string) float64
	AsString(name string) string
	// AsBytes result must not be modified.
	AsBytes(name string) []byte
	AsBool(name string) bool

	// Composite types

	AsValue(name string) ISSEValue
	AsValueIdx(idx int) ISSEValue
	// Len returns the number of elements that can be accessed by AsValueIdx.
	Len() int
}

type IReleasable interface {
	Release()
}
