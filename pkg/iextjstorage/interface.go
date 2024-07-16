/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package iextstgengine

import (
	"context"
)

type IFactory interface {
	// Returns extName => IExtStorage
	// Do NOT panic
	New(pkgPath string, modulePath string, extNames []string) (map[string]IJStorage, error)
}

// @ConcurrentAccess
type IJStorage interface {
	IReleasable
	// Do NOT panic
	Get(ctx context.Context, pk, cc IJKey) (v IJCompositeRow, ok bool, err error)
	// Do NOT panic
	Read(ctx context.Context, pk, cc IJKey, cb func(IJCompositeRow) bool) error
}

type IJKey interface {
	Pkg() string
	Name() string
	Key() IJBasicRow
}

type IJBasicRow interface {
	IReleasable
	// Do NOT panic
	AsInt64(name string) (value int64, ok bool)
	// Do NOT panic
	AsFloat64(name string) (value float64, ok bool)
	// Do NOT panic
	AsBytes(name string, value *[]byte) (ok bool)
}

type IJCompositeRow interface {
	IJBasicRow

	// FieldNames(cb func(appdef.FieldName))
	AsRow(name string) IJCompositeRow

	// Working with arrays

	ByIdx(idx int) IJCompositeRow
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
