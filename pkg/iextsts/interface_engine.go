/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package iextsts

import (
	"context"
)

// ************************************************************
// `STS` stands for `State Storage Engine`.

// Shall be created once per VVM
type ISTSEngine interface {

	// Shall be called once per storageModulePath, it effectively means that it is called once per [application, version].
	NewAppFactory(storageModulePath string, version string) (ISTSAppFactory, error)
}

type Config struct {
	Logger ISTSLogger
}

type ISTSLogger interface {
	Error(args ...interface{})
	Warning(args ...interface{})
	Info(args ...interface{})
	Verbose(args ...interface{})
}

// Use: VersionConfigPtr <load VersionConfig from json> SetConfig {NewPartitionFactory}
type ISTSAppFactory interface {
	IReleasable

	// nil means "no settings".
	// VersionConfigPtr shall be loaded from json prior calling SetConfig.
	VersionConfigPtr() *any

	// Shall be called once prior any call to the NewPartitionFactory() and after the VersionConfigPtr() result is loaded.
	SetConfig(cfg *Config) error

	// NewPartitionFactory is called once per partition per application version.
	// Existing is nil for the first partition.
	NewPartitionFactory(partitionID int, existing ISTSPartitionFactory) ISTSPartitionFactory
}

type ISTSPartitionFactory interface {
	IReleasable

	// Shall be called when a new state storage extension instance is needed (e.g. for every command/query processing)
	NewStateInstance(WSID uint64) ISTSStateInstance
}

// ************************************************************
// ISTS* interfaces

// Will be type-asserted to ISTSWith* interfaces.
type ISTSStateInstance interface {
	IReleasable
}

type ISTSWithGet interface {
	// Must return within one second.
	Get(ctx context.Context, key ISTSKey) (v ISTSRow, ok bool, err error)
}

type ISTSWithPut interface {
	// Shall return within one second.
	Put(ctx context.Context, key ISTSKey, value ISTSRow) error
}

// type ISTSWithRead interface {
// 	// key can be a partial key (filled from left to right).
// 	// go 1.23
// 	Read(ctx context.Context, key ISTSKey, cb func(ISTSRow) bool) error
// }

// ************************************************************

// As* methods panic if the requested type is not compatible with the value or the value is missing.
type ISTSKey interface {
	Namespace() string
	Name() string
	AsInt64(name string) (value int64)
	AsString(name string) (value string)
}

// ISTSRow is a read-only interface.
// As* methods panics if the requested type is not compatible with the value or the `name` parameter is invalid.
// If the `name` is valid, but the value is missing, As* methods returns a zero value.
type ISTSRow interface {
	IReleasable

	// Basic types

	AsInt64(name string) int64
	AsFloat64(name string) float64
	AsString(name string) string
	// AsBytes result must not be modified.
	AsBytes(name string) []byte
	AsBool(name string) bool

	// Composite types

	AsValue(name string) ISTSRow
	AsValueIdx(idx int) ISTSRow
	// Len returns the number of elements that can be accessed by AsValueIdx.
	Len() int
}

type IReleasable interface {
	Release()
}
