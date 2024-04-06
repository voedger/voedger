/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package exttinygo

import (
	"unsafe"

	"github.com/voedger/voedger/pkg/exttinygo/internal"
	safe "github.com/voedger/voedger/pkg/state/isafeapi"
)

func Assert(condition bool, msg string) {
	if !condition {
		Panic("assertion failed: " + msg)
	}
}

func Panic(msg string) {
	internal.HostPanic(uint32(uintptr(unsafe.Pointer(unsafe.StringData(msg)))), uint32(len(msg)))
}

func keyBuilderImpl(storage, entity string) (b TKeyBuilder) {
	return TKeyBuilder(internal.StateAPI.KeyBuilder(storage, entity))
}

func queryValueImpl(key TKeyBuilder) (value TValue, exists bool) {
	v, exsts := internal.StateAPI.QueryValue(safe.TKeyBuilder(key))
	return TValue(v), exsts
}

func mustGetValueImpl(key TKeyBuilder) TValue {
	return TValue(internal.StateAPI.MustGetValue(safe.TKeyBuilder(key)))
}

var readCallback func(key TKey, value TValue)

var safeReadCallback = func(key safe.TKey, value safe.TValue) {
	readCallback(TKey(key), TValue(value))
}

func readValuesImpl(key TKeyBuilder, cb func(key TKey, value TValue)) {
	readCallback = cb
	internal.StateAPI.ReadValues(safe.TKeyBuilder(key), safeReadCallback)
}

func updateValueImpl(key TKeyBuilder, existingValue TValue) TIntent {
	return TIntent(internal.StateAPI.UpdateValue(safe.TKeyBuilder(key), safe.TValue(existingValue)))
}

func newValueImpl(key TKeyBuilder) TIntent {
	return TIntent(internal.StateAPI.NewValue(safe.TKeyBuilder(key)))
}
