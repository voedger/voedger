/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package iextobjstorage

import (
	"context"
)

// Factory loads and returns storages specified by stNames from a location specified by locationPath
// Do NOT panic
type Factory func(locationPath string, stNames []string) (map[string]IObjectStorage, error)

// @ConcurrentAccess
type IObjectStorage interface {
	IReleasable
	// Do NOT panic
	Get(ctx context.Context, key IObjectKey) (v ICompositeObject, ok bool, err error)

	// key can be a partial key (filled from left to right)
	// Do NOT panic
	Read(ctx context.Context, key IObjectKey, cb func(ICompositeObject) bool) error
}

type IObjectKey interface {
	LocalPkg() string
	Name() string
	Key() IBasicRow
}

type IBasicRow interface {

	// Do NOT panic
	AsInt64(name string) (value int64, ok bool)
	// Do NOT panic
	AsFloat64(name string) (value float64, ok bool)

	// ??? Do NOT panic
	AsBool(name string) (value bool, ok bool)

	// Do NOT panic
	AsBytes(name string, value *[]byte) (ok bool)
}

type IBasicObject interface {
	IReleasable
	IBasicRow
}

type ICompositeObject interface {
	IBasicObject

	// FieldNames(cb func(appdef.FieldName))

	// Do NOT panic
	AsObject(name string) (value ICompositeObject, ok bool)

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
