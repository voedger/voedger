/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package iextobjstorage

import (
	"context"
)

// Do NOT panic
type Factory func(pkgPath string, modulePath string, extNames []string) (map[string]IObjectStorage, error)

// @ConcurrentAccess
type IObjectStorage interface {
	IReleasable
	// Do NOT panic
	Get(ctx context.Context, pk, cc IObjectKey) (v ICompositeObject, ok bool, err error)
	// Do NOT panic
	Read(ctx context.Context, pk, cc IObjectKey, cb func(ICompositeObject) bool) error
}

type IObjectKey interface {
	Pkg() string
	Name() string
	Key() IBasicObject
}

type IBasicObject interface {
	IReleasable
	// Do NOT panic
	AsInt64(name string) (value int64, ok bool)
	// Do NOT panic
	AsFloat64(name string) (value float64, ok bool)
	// Do NOT panic
	AsBytes(name string, value *[]byte) (ok bool)
}

type ICompositeObject interface {
	IBasicObject

	// FieldNames(cb func(appdef.FieldName))

	// Do NOT panic
	AsRow(name string) (value ICompositeObject, ok bool)

	// Working with arrays

	// Do NOT panic
	ByIdx(idx int) (value ICompositeObject, ok bool)

	// Do NOT panic
	Length() int

	// GetAsString(index int) string
	// GetAsBytes(index int) []byte
	// GetAsInt32(index int) int32
	// GetAsInt64(index int) int64
	// GetAsFloat32(index int) float32
	// GetAsFloat64(index int) float64

	// // GetAsQName(index int) appdef.QName

	// GetAsBool(index int) bool

}

type IReleasable interface {
	Release()
}
