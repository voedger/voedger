/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package exttinygo

import (
	"unsafe"

	"github.com/voedger/voedger/pkg/exttinygo/internal"
	safe "github.com/voedger/voedger/pkg/state/isafestate"
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
	return TKeyBuilder(internal.State.KeyBuilder(storage, entity))
}

func queryValueImpl(key TKeyBuilder) (value TValue, exists bool) {
	v, exsts := internal.State.QueryValue(safe.TKeyBuilder(key))
	return TValue(v), exsts
}

func mustGetValueImpl(key TKeyBuilder) TValue {
	return TValue(internal.State.MustGetValue(safe.TKeyBuilder(key)))
}

var readCallback func(key TKey, value TValue)

var safeReadCallback = func(key safe.TKey, value safe.TValue) {
	readCallback(TKey(key), TValue(value))
}

func readValuesImpl(key TKeyBuilder, cb func(key TKey, value TValue)) {
	readCallback = cb
	internal.State.ReadValues(safe.TKeyBuilder(key), safeReadCallback)
}

func updateValueImpl(key TKeyBuilder, existingValue TValue) TIntent {
	return TIntent(internal.State.UpdateValue(safe.TKeyBuilder(key), safe.TValue(existingValue)))
}

func newValueImpl(key TKeyBuilder) TIntent {
	return TIntent(internal.State.NewValue(safe.TKeyBuilder(key)))
}
