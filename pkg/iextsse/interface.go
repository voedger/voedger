/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package iextrowstorage

import (
	"context"
)

// ************************************************************
// `SSE` stands for `State Storage Extension`.

// Shall be created once per VVM
type IMainFactory interface {
	// One per instance.
	// nil means "no settings".
	// Shall be loaded once from JSON prior calling SetConfig.
	SettingsPtr() *any

	// Shall be called once prior any call to the New() and after the SettingsPtr() result is loaded.
	SetConfig(cfg *Config) error

	// Shall be called once per [application, version].
	New(storageModulePath string, version string) (IAppSSEFactory, error)
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

type IAppSSEFactory interface {
	IReleasable
	New(partitionID int) IPartitionSSEFactory
}

type IPartitionSSEFactory interface {
	IReleasable

	// Shall be called when a new state storage extension is needed (e.g. for every command/query processing)
	NewSSE(WSID uint64) ISSE
}

// ************************************************************
// ISSE* interfaces

// Will be type-asserted to ISSEWith* interfaces.
type ISSE interface {

	// Shall be called when the storage (particular [application, version]) is no longer needed.
	IReleasable
}

type ISSEWithGet interface {
	// Shall return within one second.
	Get(ctx context.Context, key ISSEKey) (v ISSECompositeRow, ok bool, err error)
}

type ISSEWithPut interface {
	// Shall return within one second.
	Put(ctx context.Context, key ISSEKey, value ISSECompositeRow) error
}

type ISSEWithRead interface {
	// key can be a partial key (filled from left to right).
	Read(ctx context.Context, key ISSEKey, cb func(ISSECompositeRow) bool) error
}

// ************************************************************

type ISSEKey interface {
	LocalPkg() string
	Name() string
	Key() ISSEKeyFields
}

type ISSEKeyFields interface {
	AsInt64(name string) (value int64, ok bool)
	AsString(name string) (value string, ok bool)
	AsBytes(name string, value *[]byte) (ok bool)
}

type ISSEBasicRowFields interface {
	AsInt64(name string) (value int64, ok bool)

	AsFloat64(name string) (value float64, ok bool)

	AsBool(name string) (value bool, ok bool)

	AsBytes(name string, value *[]byte) (ok bool)
}

type ISSECompositeRowFields interface {
	AsRowFields(name string) (value ISSERowFields, ok bool)

	AsRowFieldsAt(idx int) (value ISSERowFields, ok bool)

	// Returns value >= 0
	Length() int
}

type ISSERowFields interface {
	ISSEBasicRowFields
	ISSECompositeRowFields
}

type ISSEBasicRow interface {
	IReleasable
	ISSEBasicRowFields
}

type ISSECompositeRow interface {
	IReleasable
	ISSEBasicRowFields
	ISSECompositeRowFields
}

type IReleasable interface {
	Release()
}
