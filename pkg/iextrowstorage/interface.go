/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package iextrowstorage

import (
	"context"
)

type IFactory interface {
	// Do NOT panic
	// New is called once per [application, package, version]
	// New loads and returns storages specified by stNames from a location specified by locationPath
	New(cgd *Config) (map[string]IRowStorage, error)
}

type Config struct {
	LocationPath string
	StorageNames []string
	Logger       IRSLogger
}

type IRSLogger interface {
	Error(args ...interface{})
	Warning(args ...interface{})
	Info(args ...interface{})
	Verbose(args ...interface{})
}

// @ConcurrentAccess
// Will be type-asserted to IRowStorageWith* interfaces
type IRowStorage interface {
	IReleasable
}

type IRowStorageWithGet interface {
	// Do NOT panic
	Get(ctx context.Context, key IRowKey) (v ICompositeRow, ok bool, err error)
}

type IRowStorageWithPut interface {
	// Do NOT panic
	Put(ctx context.Context, key IRowKey, value ICompositeRow) error
}

type IRowStorageWithRead interface {
	// key can be a partial key (filled from left to right)
	// Do NOT panic
	Read(ctx context.Context, key IRowKey, cb func(ICompositeRow) bool) error
}

type IRowKey interface {
	LocalPkg() string
	Name() string
	Key() IBasicRowFields
}

type IBasicRowFields interface {

	// Do NOT panic
	AsInt64(name string) (value int64, ok bool)
	// Do NOT panic
	AsFloat64(name string) (value float64, ok bool)

	// ??? Do NOT panic
	AsBool(name string) (value bool, ok bool)

	// Do NOT panic
	AsBytes(name string, value *[]byte) (ok bool)
}

type IBasicRow interface {
	IReleasable
	IBasicRowFields
}

type ICompositeRow interface {
	IBasicRow

	// FieldNames(cb func(appdef.FieldName))

	// Do NOT panic
	AsRow(name string) (value ICompositeRow, ok bool)

	// Working with arrays

	// Do NOT panic
	ByIdx(idx int) (value ICompositeRow, ok bool)

	// Do NOT panic
	Length() int
}

type IReleasable interface {
	Release()
}
