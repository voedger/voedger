/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package iextenginewasm

import (
	"context"
	"testing"
	"time"

	"github.com/heeus/wazero/api"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/state"
)

var limits = iextengine.ExtensionLimits{
	ExecutionInterval: 100 * time.Second,
}

var extIO = &mockIo{}

func Test_BasicUsage(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()

	moduleUrl := testModuleURL("./_testdata/basicusage/pkg.wasm")
	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{})
	if err != nil {
		panic(err)
	}
	defer closer()

	extensions := make(map[string]iextengine.IExtension)
	extEngine.ForEach(func(name string, ext iextengine.IExtension) {
		extensions[name] = ext
	})
	require.Equal(2, len(extensions))
	extEngine.SetLimits(limits)

	//
	// Invoke command
	//
	require.NoError(extensions["exampleCommand"].Invoke(extIO))
	require.Equal(1, len(extIO.intents))
	v := extIO.intents[0].value.(*mockValueBuilder)

	require.Equal("test@gmail.com", v.items["from"])
	require.Equal("email@user.com", v.items["to"])
	require.Equal("You are invited", v.items["body"])

	//
	// Invoke projector which parses JSON
	//
	extIO = &mockIo{}    // reset intents
	projectorMode = true // state will return different Event
	require.NoError(extensions["updateSubscriptionProjector"].Invoke(extIO))

	require.Equal(1, len(extIO.intents))
	v = extIO.intents[0].value.(*mockValueBuilder)

	require.Equal("test@gmail.com", v.items["from"])
	require.Equal("customer@test.com", v.items["to"])
	require.Equal("Your subscription has been updated. New status: active", v.items["body"])
}

func requireMemStat(t *testing.T, wasmEngine *wazeroExtEngine, mallocs, frees, heapInUse uint32) {
	m, err := wasmEngine.getMallocs()
	require.NoError(t, err)
	f, err := wasmEngine.getFrees()
	require.NoError(t, err)
	h, err := wasmEngine.getHeapinuse()
	require.NoError(t, err)

	require.Equal(t, mallocs, uint32(m))
	require.Equal(t, frees, uint32(f))
	require.Equal(t, heapInUse, uint32(h))
}

func requireMemStatEx(t *testing.T, wasmEngine *wazeroExtEngine, mallocs, frees, heapSys, heapInUse uint32) {
	requireMemStat(t, wasmEngine, mallocs, frees, heapInUse)
	h, err := wasmEngine.getHeapSys()
	require.NoError(t, err)
	require.Equal(t, heapSys, uint32(h))
}

func Test_Allocs_ManualGC(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/allocs/pkggc.wasm")
	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{})
	require.NoError(err)
	defer closer()

	extensions := extractExtensions(extEngine)
	extAppend := extensions["arrAppend"]
	extReset := extensions["arrReset"]
	wasmEngine := extEngine.(*wazeroExtEngine)

	const expectedHeapSize = uint32(1999536)

	requireMemStatEx(t, wasmEngine, 1, 0, expectedHeapSize, WasmPreallocatedBufferSize)
	wasmEngine.SetLimits(limits)

	require.NoError(extAppend.Invoke(extIO))
	requireMemStatEx(t, wasmEngine, 3, 0, expectedHeapSize, WasmPreallocatedBufferSize+32)

	require.NoError(extAppend.Invoke(extIO))
	requireMemStatEx(t, wasmEngine, 5, 0, expectedHeapSize, WasmPreallocatedBufferSize+64)

	require.NoError(extAppend.Invoke(extIO))
	requireMemStatEx(t, wasmEngine, 7, 0, expectedHeapSize, WasmPreallocatedBufferSize+6*16)

	require.NoError(extReset.Invoke(extIO))
	requireMemStatEx(t, wasmEngine, 7, 0, expectedHeapSize, WasmPreallocatedBufferSize+6*16)

	require.NoError(wasmEngine.gc())
	requireMemStatEx(t, wasmEngine, 7, 6, expectedHeapSize, WasmPreallocatedBufferSize)
}

func Test_Allocs_AutoGC(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/allocs/pkggc.wasm")
	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: 0xffff})
	require.NoError(err)
	defer closer()

	extensions := extractExtensions(extEngine)

	const expectedHeapSize = uint32(1999536)
	var expectedAllocs = uint32(1)
	var expectedFrees = uint32(0)
	extAppend := extensions["arrAppend"]
	extReset := extensions["arrReset"]
	wasmEngine := extEngine.(*wazeroExtEngine)

	requireMemStatEx(t, wasmEngine, expectedAllocs, expectedFrees, expectedHeapSize, WasmPreallocatedBufferSize)

	defer wasmEngine.close()

	calculatedHeapInUse := uint32(WasmPreallocatedBufferSize)
	for calculatedHeapInUse < expectedHeapSize-16 {
		require.NoError(extAppend.Invoke(extIO))
		require.NoError(extReset.Invoke(extIO))
		calculatedHeapInUse += 32
		expectedAllocs += 2
	}

	// no GC has been called, and the heap is now full
	requireMemStatEx(t, wasmEngine, expectedAllocs, expectedFrees, expectedHeapSize, expectedHeapSize-16)

	// next call will trigger auto-GC
	require.NoError(extAppend.Invoke(extIO))
	expectedAllocs += 2
	expectedFrees += expectedAllocs - 9
	requireMemStat(t, wasmEngine, expectedAllocs, expectedFrees, WasmPreallocatedBufferSize+128)

	// next call will not trigger auto-GC
	require.NoError(extReset.Invoke(extIO))
	requireMemStat(t, wasmEngine, expectedAllocs, expectedFrees, WasmPreallocatedBufferSize+128) // stays the same

}

func Test_NoGc_MemoryOverflow(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()

	moduleUrl := testModuleURL("./_testdata/allocs/pkg.wasm")
	extEngine, _, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: 0x10})
	require.ErrorContains(err, "the minimum limit of memory is: 1700000.0 bytes, requested limit is: 1048576.0")
	require.Nil(extEngine)

	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20})
	require.NoError(err)
	defer closer()

	extensions := extractExtensions(extEngine)
	extAppend := extensions["arrAppend"]
	extReset := extensions["arrReset"]
	wasmEngine := extEngine.(*wazeroExtEngine)

	var expectedAllocs = uint32(1)
	var expectedFrees = uint32(0)

	requireMemStatEx(t, wasmEngine, expectedAllocs, expectedFrees, uint32(WasmPreallocatedBufferSize), uint32(WasmPreallocatedBufferSize))

	calculatedHeapInUse := WasmPreallocatedBufferSize
	err = nil
	for calculatedHeapInUse < 0x20*iextengine.MemoryPageSize {
		err = wasmEngine.invoke(extAppend.(*wasmExtension).f, extIO)
		if err != nil {
			break
		}
		err = wasmEngine.invoke(extReset.(*wasmExtension).f, extIO)
		if err != nil {
			break
		}
		calculatedHeapInUse += 32
	}
	require.ErrorContains(err, "alloc")
}

func Test_SetLimitsExecutionInterval(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/allocs/pkg.wasm")
	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{})
	require.NoError(err)
	defer closer()
	extensions := extractExtensions(extEngine)

	maxDuration := time.Millisecond * 50
	extEngine.SetLimits(iextengine.ExtensionLimits{
		ExecutionInterval: maxDuration,
	})
	t0 := time.Now()
	err = extensions["longFunc"].Invoke(extIO)

	require.ErrorIs(err, api.ErrDuration)
	require.Less(time.Since(t0), maxDuration*2)
}

func Test_HandlePanics(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/panics/pkg.wasm")
	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20})
	require.NoError(err)
	defer closer()

	extensions := extractExtensions(extEngine)
	err = extensions["incorrectStorageQname"].Invoke(extIO)
	// full text is: "error running extension [incorrectStorageQname]: invalid string representation of qualified name: foo (recovered by wazero)\nwasm stack trace:\n\tenv.HostGetKey(i32,i32,i32,i32) i64\n\t.incorrectStorageQname()\n\t.incorrectStorageQname.command_export()"
	require.ErrorContains(err, "invalid string representation of qualified name: foo")

	tf := func(name string, expect string) {
		e := extensions[name].Invoke(extIO)
		require.ErrorContains(e, expect)
	}

	tf("incorrectEntityQname", "invalid string representation of qualified name: abc")
	tf("unsupportedStorage", "unsupported storage")
	tf("incorrectKeyBuilder", PanicIncorrectKeyBuilder)
	tf("canExistIncorrectKey", PanicIncorrectKeyBuilder)
	tf("mustExistIncorrectKey", PanicIncorrectKeyBuilder)
	tf("readIncorrectKeyBuilder", PanicIncorrectKeyBuilder)
	tf("incorrectKey", PanicIncorrectKey)
	tf("incorrectValue", PanicIncorrectValue)
	tf("incorrectValue2", PanicIncorrectValue)
	tf("incorrectValue3", PanicIncorrectValue)
	tf("mustExist", state.ErrNotExists.Error())
	tf("incorrectKeyBuilderOnNewValue", PanicIncorrectKeyBuilder)
	tf("incorrectKeyBuilderOnUpdateValue", PanicIncorrectKeyBuilder)
	tf("incorrectValueOnUpdateValue", PanicIncorrectValue)
	tf("incorrectIntentId", PanicIncorrectIntent)
	tf("readPanic", PanicIncorrectValue)
	tf("readError", errTestIOError.Error())
	tf("getError", errTestIOError.Error())
	tf("queryError", errTestIOError.Error())
	tf("newValueError", errTestIOError.Error())
	tf("updateValueError", errTestIOError.Error())
	tf("asStringMemoryOverflow", "alloc")

}

func Test_QueryValue(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/tests/pkg.wasm")
	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20})
	require.NoError(err)
	defer closer()

	extensions := extractExtensions(extEngine)
	err = extensions["testQueryValue"].Invoke(extIO)
	require.NoError(err)
}

func Test_RecoverEngine(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/allocs/pkg.wasm")
	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20})
	require.NoError(err)
	defer closer()
	extensions := extractExtensions(extEngine)
	extAppend := extensions["arrAppend2"]

	require.Nil(extAppend.Invoke(extIO))
	require.Nil(extAppend.Invoke(extIO))
	require.Nil(extAppend.Invoke(extIO))
	require.NotNil(extAppend.Invoke(extIO))

	require.Nil(extAppend.Invoke(extIO))
	require.Nil(extAppend.Invoke(extIO))
	require.Nil(extAppend.Invoke(extIO))
	require.NotNil(extAppend.Invoke(extIO))
}

func extractExtensions(engine iextengine.IExtensionEngine) map[string]iextengine.IExtension {
	extensions := make(map[string]iextengine.IExtension)
	engine.ForEach(func(name string, ext iextengine.IExtension) {
		extensions[name] = ext
	})
	return extensions
}

func Test_Read(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/tests/pkg.wasm")
	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{})
	require.NoError(err)
	defer closer()
	extensions := extractExtensions(extEngine)

	err = extensions["testRead"].Invoke(extIO)
	require.NoError(err)

}

func Test_AsBytes(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/tests/pkg.wasm")
	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{})
	require.NoError(err)
	defer closer()
	wasmEngine := extEngine.(*wazeroExtEngine)
	extensions := extractExtensions(extEngine)
	requireMemStatEx(t, wasmEngine, 1, 0, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)
	err = extensions["asBytes"].Invoke(extIO)
	require.NoError(err)
	requireMemStatEx(t, wasmEngine, 2, 0, WasmPreallocatedBufferSize+2000000, WasmPreallocatedBufferSize+2000000)
}

func Test_AsBytesOverflow(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/tests/pkg.wasm")
	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20})
	require.NoError(err)
	defer closer()
	wasmEngine := extEngine.(*wazeroExtEngine)
	extensions := extractExtensions(extEngine)
	requireMemStatEx(t, wasmEngine, 1, 0, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)
	err = extensions["asBytes"].Invoke(extIO)
	require.ErrorContains(err, "alloc")
}

func Test_NoAllocs(t *testing.T) {

	extIO = &mockIo{}
	projectorMode = false
	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/tests/pkg.wasm")
	extEngine, closer, err := ExtEngineWazeroFactory(ctx, moduleUrl, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20})
	require.NoError(err)
	defer closer()

	wasmEngine := extEngine.(*wazeroExtEngine)
	extensions := extractExtensions(extEngine)

	requireMemStatEx(t, wasmEngine, 1, 0, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)

	err = extensions["testNoAllocs"].Invoke(extIO)
	require.NoError(err)

	requireMemStatEx(t, wasmEngine, 1, 0, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)

	require.Equal(2, len(extIO.intents))
	v0 := extIO.intents[0].value.(*mockValueBuilder)

	require.Equal("test@gmail.com", v0.items["from"])
	require.Equal(int32(668), v0.items["port"])
	bytes := (v0.items["key"]).([]byte)
	require.Equal(5, len(bytes))

	v1 := extIO.intents[1].value.(*mockValueBuilder)
	require.Equal(int32(12346), v1.items["offs"])
	require.Equal("sys.InvitationAccepted", v1.items["qname"])

}
