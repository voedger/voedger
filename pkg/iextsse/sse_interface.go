/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package iextsse

import (
	"context"
)

// ************************************************************
// `SSE` stands for `State Storage Extension`.

// Shall be created once per VVM
type ISSEVvmFactory interface {

	// Shall be called once per storageModulePath, it effectively means that it is called once per [application, version].
	NewAppFactory(storageModulePath string, version string) (ISSEAppVerFactory, error)
}

type Config struct {
	Logger ISSELogger
}

type ISSELogger interface {
	Error(args ...interface{})
	Warning(args ...interface{})
	Info(args ...interface{})
	Verbose(args ...interface{})
}

// Use: VersionConfigPtr <load VersionConfig from json> SetConfig {NewPartitionFactory}
type ISSEAppVerFactory interface {
	// Released when there are no more partitions for the application version.
	IReleasable

	// nil means "no settings".
	// VersionConfigPtr shall be loaded from a json config  (if the config exists) prior calling SetConfig.
	VersionConfigPtr() *any

	// Shall be called once prior any call to the NewPartitionFactory() and after the VersionConfigPtr() result is loaded.
	SetConfig(cfg *Config) error

	// NewPartitionFactory is called once per partition per application version.
	// Existing is nil for the first partition.
	NewPartitionFactory(partitionID PartitionID, existing ISSEPartitionFactory) ISSEPartitionFactory
}

type ISSEPartitionFactory interface {
	// Released when the partition factory is not the current factory and there are no more active StateStorage-s for the partition factory.
	IReleasable

	// Shall be called when a new state storage extension instance is needed (e.g. for every command/query processing)
	NewStateStorage(WSID uint64) ISSEStateStorage
}

// ************************************************************
// ISSE* interfaces

// Will be type-asserted to ISSEWith* interfaces.
type ISSEStateStorage interface {
	IReleasable
}

type ISSEWithGet interface {
	// Must return within one second.
	Get(ctx context.Context, key ISSEKey) (v ISSERow, ok bool, err error)
}

type ISSEWithPut interface {
	// Shall return within one second.
	Put(ctx context.Context, key ISSEKey, value ISSERow) error
}

// type ISSEWithRead interface {
// 	// key can be a partial key (filled from left to right).
// 	// go 1.23
// 	Read(ctx context.Context, key ISSEKey, cb func(ISSERow) bool) error
// }

// ************************************************************

// As* methods panic if the requested type is not compatible with the value or the value is missing.
type ISSEKey interface {
	Namespace() string
	Name() string
	AsInt64(name string) (value int64)
	AsString(name string) (value string)
}

// ISSERow is a read-only interface.
// As* methods panics if the requested type is not compatible with the value or the `name` parameter is invalid.
// If the `name` is valid, but the value is missing, As* methods returns a zero value.
type ISSERow interface {
	IReleasable

	// Basic types

	AsInt64(name string) int64
	AsFloat64(name string) float64
	AsString(name string) string
	// AsBytes result must not be modified.
	AsBytes(name string) []byte
	AsBool(name string) bool

	// Composite types

	AsValue(name string) ISSERow
	AsValueIdx(idx int) ISSERow
	// Len returns the number of elements that can be accessed by AsValueIdx.
	Len() int
}

type IReleasable interface {
	Release()
}
