/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package exttinygo

import (
	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
)

type TKeyBuilder safe.TKeyBuilder
type TIntent safe.TIntent
type TValue safe.TValue
type TKey safe.TKey
type QName safe.QName

var KeyBuilder func(storage, entity string) (b TKeyBuilder) = keyBuilderImpl

// QueryValue queries value. When not exists it returns exists=false and value=nil.
var QueryValue func(key TKeyBuilder) (value TValue, exists bool) = queryValueImpl

// MustGetValue gets value. Panics when value is not exist
var MustGetValue func(key TKeyBuilder) TValue = mustGetValueImpl

// ReadValues reads using partial key and returns values in callback.
//
// Important: key and value are not kept after callback!
var ReadValues func(key TKeyBuilder, callback func(key TKey, value TValue)) = readValuesImpl

// UpdateValue creates intent to update a value
var UpdateValue func(key TKeyBuilder, existingValue TValue) TIntent = updateValueImpl

// NewValue creates intent for new value
var NewValue func(key TKeyBuilder) TIntent = newValueImpl

/*
type IKey interface {
	AsString(name string) string
	AsInt32(name string) int32
	AsInt64(name string) int64
	AsFloat32(name string) float32
	AsFloat64(name string) float64
	AsBytes(name string) []byte
	AsQName(name string) QName
	AsBool(name string) bool
}

type IValue interface {
	Len() int

	AsString(name string) string
	AsBytes(name string) []byte
	AsInt32(name string) int32
	AsInt64(name string) int64
	AsFloat32(name string) float32
	AsFloat64(name string) float64
	AsQName(name string) QName
	AsBool(name string) bool
	AsValue(name string) IValue // throws panic if field is not an object or array

	GetAsString(index int) string
	GetAsBytes(index int) []byte
	GetAsInt32(index int) int32
	GetAsInt64(index int) int64
	GetAsFloat32(index int) float32
	GetAsFloat64(index int) float64
	GetAsQName(index int) QName
	GetAsBool(index int) bool
	GetAsValue(index int) IValue // throws panic if field is not an object or array
}

type IRowWriter interface {
	PutInt32(name string, value int32)
	PutInt64(name string, value int64)
	PutFloat32(name string, value float32)
	PutFloat64(name string, value float64)
	PutString(name string, value string)
	PutBytes(name string, value []byte)
	PutQName(name string, value QName)
	PutBool(name string, value bool)
}

type IKeyBuilder interface {
	IRowWriter
}

type IIntent interface {
	IRowWriter
}
*/
