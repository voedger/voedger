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
type ISSEVVMFactory interface {
	// One per instance.
	// nil means "no settings".
	// Shall be loaded once from JSON prior calling SetConfig.
	SettingsPtr() *any

	// Shall be called once prior any call to the New() and after the SettingsPtr() result is loaded.
	SetConfig(cfg *Config) error

	// Shall be called once per storageModulePath, it effectively means that it is called once per [application, version].
	NewAppFactory(storageModulePath string, version string) (ISSEAppFactory, error)
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

type ISSEAppFactory interface {
	IReleasable
	NewPartitionFactory(partitionID int) ISSEPartitionFactory
}

type ISSEPartitionFactory interface {
	IReleasable

	// Shall be called when a new state storage extension instance is needed (e.g. for every command/query processing)
	NewStateInstance(WSID uint64) ISSEStateInstance
}

// ************************************************************
// ISSE* interfaces

// Will be type-asserted to ISSEWith* interfaces.
type ISSEStateInstance interface {
	IReleasable
}

type ISSEWithGet interface {
	// Shall return within one second.
	Get(ctx context.Context, key ISSEKey) (v ISSEValue, ok bool, err error)
}

type ISSEWithPut interface {
	// Shall return within one second.
	Put(ctx context.Context, key ISSEKey, value ISSEValue) error
}

type ISSEWithRead interface {
	// key can be a partial key (filled from left to right).
	Read(ctx context.Context, key ISSEKey, cb func(ISSEValue) bool) error
}

// ************************************************************

type ISSEKey interface {
	Namespace() string
	Name() string
	AsInt64(name string) (value int64, ok bool)
	AsString(name string) (value string, ok bool)
}

type ISSEValue interface {
	AsInt64(name string) (value int64, ok bool)
	AsFloat64(name string) (value float64, ok bool)
	AsString(name string) (value string, ok bool)
	AsBytes(name string, bytes *[]byte) (ok bool)
	AsBool(name string) (value bool, ok bool)
	AsValue(name string) (value ISSEValue, ok bool)
	AsValueAt(idx int) (value ISSEValue, ok bool)
	Len() int
	IReleasable
}

type IReleasable interface {
	Release()
}
