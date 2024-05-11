/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package exttinygo

import (
	"github.com/voedger/voedger/pkg/exttinygo/internal"
	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
)

func Assert(condition bool, msg string) {
	if !condition {
		panic("assertion failed: " + msg)
	}
}

func keyBuilderImpl(storage, entity string) (b TKeyBuilder) {
	return TKeyBuilder(internal.SafeStateAPI.KeyBuilder(storage, entity))
}

func queryValueImpl(key TKeyBuilder) (value TValue, exists bool) {
	v, exsts := internal.SafeStateAPI.QueryValue(safe.TKeyBuilder(key))
	return TValue(v), exsts
}

func mustGetValueImpl(key TKeyBuilder) TValue {
	return TValue(internal.SafeStateAPI.MustGetValue(safe.TKeyBuilder(key)))
}

var readCallback func(key TKey, value TValue)

var safeReadCallback = func(key safe.TKey, value safe.TValue) {
	readCallback(TKey(key), TValue(value))
}

func readValuesImpl(key TKeyBuilder, cb func(key TKey, value TValue)) {
	readCallback = cb
	internal.SafeStateAPI.ReadValues(safe.TKeyBuilder(key), safeReadCallback)
}

func updateValueImpl(key TKeyBuilder, existingValue TValue) TIntent {
	return TIntent(internal.SafeStateAPI.UpdateValue(safe.TKeyBuilder(key), safe.TValue(existingValue)))
}

func newValueImpl(key TKeyBuilder) TIntent {
	return TIntent(internal.SafeStateAPI.NewValue(safe.TKeyBuilder(key)))
}
