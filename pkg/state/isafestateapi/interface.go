/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */

package isafestateapi

type TKeyBuilder int64
type TKey int64
type TValue int64
type TIntent int64
type QName struct {
	FullPkgName string
	Entity      string
}

type IStateSafeAPI interface {
	// Basic functions
	KeyBuilder(storage, entityFullQname string) TKeyBuilder
	MustGetValue(key TKeyBuilder) TValue
	QueryValue(key TKeyBuilder) (value TValue, ok bool)
	NewValue(key TKeyBuilder) (v TIntent)
	UpdateValue(key TKeyBuilder, existingValue TValue) (v TIntent)
	ReadValues(key TKeyBuilder, callback func(key TKey, value TValue))

	// Key Builder
	KeyBuilderPutInt32(key TKeyBuilder, name string, value int32)
	KeyBuilderPutInt64(key TKeyBuilder, name string, value int64)
	KeyBuilderPutRecordID(key TKeyBuilder, name string, value int64)
	KeyBuilderPutFloat32(key TKeyBuilder, name string, value float32)
	KeyBuilderPutFloat64(key TKeyBuilder, name string, value float64)
	KeyBuilderPutString(key TKeyBuilder, name string, value string)
	KeyBuilderPutBytes(key TKeyBuilder, name string, value []byte)
	KeyBuilderPutQName(key TKeyBuilder, name string, value QName)
	KeyBuilderPutBool(key TKeyBuilder, name string, value bool)

	// Key
	KeyAsInt32(k TKey, name string) int32
	KeyAsInt64(k TKey, name string) int64
	KeyAsFloat32(k TKey, name string) float32
	KeyAsFloat64(k TKey, name string) float64
	KeyAsBytes(k TKey, name string) []byte
	KeyAsString(k TKey, name string) string
	KeyAsQName(k TKey, name string) QName
	KeyAsBool(k TKey, name string) bool

	// Value
	ValueAsValue(v TValue, name string) (result TValue)
	ValueAsInt32(v TValue, name string) int32
	ValueAsInt64(v TValue, name string) int64
	ValueAsFloat32(v TValue, name string) float32
	ValueAsFloat64(v TValue, name string) float64
	ValueAsBytes(v TValue, name string) []byte
	ValueAsQName(v TValue, name string) QName
	ValueAsBool(v TValue, name string) bool
	ValueAsString(v TValue, name string) string

	ValueLen(v TValue) int
	ValueGetAsValue(v TValue, index int) (result TValue)
	ValueGetAsInt32(v TValue, index int) int32
	ValueGetAsInt64(v TValue, index int) int64
	ValueGetAsFloat32(v TValue, index int) float32
	ValueGetAsFloat64(v TValue, index int) float64
	ValueGetAsBytes(v TValue, index int) []byte
	ValueGetAsQName(v TValue, index int) QName
	ValueGetAsBool(v TValue, index int) bool
	ValueGetAsString(v TValue, index int) string

	// Intent
	IntentPutInt64(v TIntent, name string, value int64)
	IntentPutInt32(v TIntent, name string, value int32)
	IntentPutFloat32(v TIntent, name string, value float32)
	IntentPutFloat64(v TIntent, name string, value float64)
	IntentPutString(v TIntent, name string, value string)
	IntentPutBytes(v TIntent, name string, value []byte)
	IntentPutQName(v TIntent, name string, value QName)
	IntentPutBool(v TIntent, name string, value bool)
}
