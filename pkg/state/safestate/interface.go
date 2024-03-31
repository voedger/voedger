/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */

package safestate

type TSafeKeyBuilder int64
type TSafeValue int64
type TSafeIntent int64

type ISafeState interface {
	// Basic functions
	KeyBuilder(storage, entityFullQname string) TSafeKeyBuilder
	MustGetValue(key TSafeKeyBuilder) TSafeValue
	QueryValue(key TSafeKeyBuilder) (value TSafeValue, ok bool)
	NewValue(key TSafeKeyBuilder) (v TSafeIntent)
	UpdateValue(key TSafeKeyBuilder, existingValue TSafeValue) (v TSafeIntent)

	// Key Builder

	KeyBuilderPutInt32(key TSafeKeyBuilder, name string, value int32)

	// Value

	ValueAsValue(v TSafeValue, name string) (result TSafeValue)
	ValueLen(v TSafeValue) int
	ValueGetAsValue(v TSafeValue, index int) (result TSafeValue)
	ValueAsInt32(v TSafeValue, name string) int32
	ValueAsInt64(v TSafeValue, name string) int64

	// Intent
	IntentPutInt64(v TSafeIntent, name string, value int64)
}
