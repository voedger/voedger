/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package iextrowstorage

import (
	"context"
)

// Shall be created once per [application, storage extension, version]
type IFactoryFactory interface {
	// One per instance
	// Shall be loaded once from JSON prior any call to the New()
	SettingsPtr() *any

	// Shall be called prior any call to the New() and after the SettingsPtr() result is loaded
	SetConfig(cfg *Config) error

	// New is called once per [application, IFactoryFactory]
	NewFactory(filePath string, version string) (IRowStorageFactory, error)
}

type Config struct {
	Logger IRSLogger
}

type IRSLogger interface {
	Error(args ...interface{})
	Warning(args ...interface{})
	Info(args ...interface{})
	Verbose(args ...interface{})
}

type IRowStorageFactory interface {
	// Once per [application, storage package, version, partition]
	// partition >= 0
	New(partition int) IRowStorage
}

// ************************************************************
// IRowStorage* interfaces

// @ConcurrentAccess
// Will be type-asserted to IRowStorageWith* interfaces
type IRowStorage interface {
	IReleasable
}

type IRowStorageWithGet interface {
	// Shall return in no more than one second.
	Get(ctx context.Context, key IRowKey) (v ICompositeRow, ok bool, err error)
}

type IRowStorageWithPut interface {
	Put(ctx context.Context, key IRowKey, value ICompositeRow) error
}

type IRowStorageWithRead interface {
	// key can be a partial key (filled from left to right)
	Read(ctx context.Context, key IRowKey, cb func(ICompositeRow) bool) error
}

// ************************************************************

type IRowKey interface {
	LocalPkg() string
	Name() string
	Key() IBasicRowFields
}

type IBasicRowFields interface {
	AsInt64(name string) (value int64, ok bool)

	AsFloat64(name string) (value float64, ok bool)

	AsBool(name string) (value bool, ok bool)

	AsBytes(name string, value *[]byte) (ok bool)
}

type ICompositeRowFields interface {
	AsRowFields(name string) (value IRowFields, ok bool)

	AsRowFieldsAt(idx int) (value IRowFields, ok bool)

	// Returns value >= 0
	Length() int
}

type IRowFields interface {
	IBasicRowFields
	ICompositeRowFields
}

type IBasicRow interface {
	IReleasable
	IBasicRowFields
}

type ICompositeRow interface {
	IReleasable
	IBasicRowFields
	ICompositeRowFields
}

type IReleasable interface {
	Release()
}
