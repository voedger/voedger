/*
* Copyright (c) 2023-present unTill Pro, Ltd.
*  @author Michael Saigachenko
 */

package exttinygo

import (
	"runtime"
	"unsafe"
)

func Assert(condition bool, msg string) {
	if !condition {
		Panic("assertion failed: " + msg)
	}
}

func Panic(msg string) {
	hostPanic(uint32(uintptr(unsafe.Pointer(unsafe.StringData(msg)))), uint32(len(msg)))
}

const maxUint = ^uint64(0)

func queryValueImpl(key TKeyBuilder) (TValue, bool) {
	id := hostQueryValue(uint64(key))
	if id != maxUint {
		return TValue(id), true
	} else {
		return TValue(0), false
	}
}

func mustGetValueImpl(key TKeyBuilder) TValue {
	return TValue(hostGetValue(uint64(key)))
}

func updateValueImpl(key TKeyBuilder, existingValue TValue) TIntent {
	return TIntent(hostUpdateValue(uint64(key), uint64(existingValue)))
}

func newValueImpl(key TKeyBuilder) TIntent {
	return TIntent(hostNewValue(uint64(key)))
}

func readValuesImpl(key TKeyBuilder, callback func(key TKey, value TValue)) {
	currentReadCallback = callback
	hostReadValues(uint64(key))
}

var currentReadCallback func(key TKey, value TValue)

//lint:ignore U1000 this is an exported func
//export WasmOnReadValue
func onReadValue(key, value uint64) {
	currentReadCallback(TKey(key), TValue(value))
}

/*
	returns 0 when not exists
*/

//lint:ignore U1000 this is an exported func
//export WasmAbiVersion_0_0_1
func proxyABIVersion() {
}

var ms runtime.MemStats

//lint:ignore U1000 this is an exported func
//export WasmGetHeapInuse
func getHeapInuse() uint64 {
	runtime.ReadMemStats(&ms)
	return ms.HeapInuse
}

//lint:ignore U1000 this is an exported func
//export WasmGetMallocs
func getMallocs() uint64 {
	runtime.ReadMemStats(&ms)
	return ms.Mallocs
}

//lint:ignore U1000 this is an exported func
//export WasmGetFrees
func getFrees() uint64 {
	runtime.ReadMemStats(&ms)
	return ms.Frees
}

//lint:ignore U1000 this is an exported func
//export WasmGetHeapSys
func getHeapSys() uint64 {
	runtime.ReadMemStats(&ms)
	return ms.HeapSys
}

//lint:ignore U1000 this is an exported func
//export WasmGC
func gc() {
	runtime.GC()
}
