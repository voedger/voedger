/*
  - Copyright (c) 2023-present unTill Software Development Group B V.
    @author Michael Saigachenko
*/

package iextenginewazero

import (
	"context"
	"embed"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero/sys"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/parser"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/state"
)

var extIO = &mockIo{}

const testPkg = "test"

func Test_BasicUsage(t *testing.T) {
	// Test Consts
	const intentsLimit = 5
	const bundlesLimit = 5
	offset := istructs.Offset(123)
	const ws = istructs.WSID(1)
	const partition = istructs.PartitionID(1)
	require := require.New(t)
	newOrderCmd := appdef.NewQName("air", "NewOrder")
	calcOrderedItemsProjector := appdef.NewQName("air", "CalcOrderedItems")
	orderedItemsView := appdef.NewQName("air", "OrderedItems")

	// Prepare app
	app := appStructsFromSQL("github.com/untillpro/airs-bp3/packages/air", `APPLICATION test(); 
		WORKSPACE Restaurant (
			TABLE Order INHERITS ODoc (
				Year int32,
				Month int32,
				Day int32,
				Waiter ref,
				Items TABLE OrderItems (
					Quantity int32,
					SinglePrice currency,
					Article ref
				)
			);
			VIEW OrderedItems (
				Year int32,
				Month int32,
				Day int32,
				Amount currency,
				PRIMARY KEY ((Year), Month, Day)
			) AS RESULT OF CalcOrderedItems;
			EXTENSION ENGINE WASM(
				COMMAND NewOrder(Order);
				PROJECTOR CalcOrderedItems AFTER EXECUTE ON NewOrder INTENTS(View(OrderedItems));
			);
		)
		`,
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(newOrderCmd, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(istructs.Projector{Name: calcOrderedItemsProjector})
		})

	// Build NewOrder event
	reb := app.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			Workspace:         ws,
			HandlingPartition: partition,
			PLogOffset:        offset,
			QName:             newOrderCmd,
		},
	})
	orderBuilder := reb.ArgumentObjectBuilder()
	orderBuilder.PutRecordID(appdef.SystemField_ID, 1)
	orderBuilder.PutInt32("Year", 2023)
	orderBuilder.PutInt32("Month", 1)
	orderBuilder.PutInt32("Day", 1)
	items := orderBuilder.ChildBuilder("Items")
	items.PutRecordID(appdef.SystemField_ID, 2)
	items.PutInt32("Quantity", 1)
	items.PutInt64("SinglePrice", 100)
	items = orderBuilder.ChildBuilder("Items")
	items.PutRecordID(appdef.SystemField_ID, 3)
	items.PutInt32("Quantity", 2)
	items.PutInt64("SinglePrice", 50)
	rawEvent, err := reb.BuildRawEvent()
	if err != nil {
		panic(err)
	}
	event, err := app.Events().PutPlog(rawEvent, nil, istructsmem.NewIDGenerator())
	if err != nil {
		panic(err)
	}

	eventFunc := func() istructs.IPLogEvent { return event }
	cudFunc := func() istructs.ICUD { return reb.CUDBuilder() }
	argFunc := func() istructs.IObject { return event.ArgumentObject() }
	unloggedArgFunc := func() istructs.IObject { return nil }
	appFunc := func() istructs.IAppStructs { return app }

	// Create states for Command processor and Actualizer
	actualizerState := state.ProvideAsyncActualizerStateFactory()(context.Background(), app, nil, state.SimpleWSIDFunc(ws), nil, nil, eventFunc, intentsLimit, bundlesLimit)
	cmdProcState := state.ProvideCommandProcessorStateFactory()(context.Background(), appFunc, nil, state.SimpleWSIDFunc(ws), nil, cudFunc, nil, nil, intentsLimit, nil, argFunc, unloggedArgFunc)

	// Create extension package from WASM
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/basicusage/pkg.wasm")
	packages := []iextengine.ExtensionPackage{
		{
			QualifiedName:  testPkg,
			ModuleUrl:      moduleUrl,
			ExtensionNames: []string{calcOrderedItemsProjector.Entity(), newOrderCmd.Entity()},
		},
	}

	// Create extension engine
	factory := ProvideExtensionEngineFactory(true)
	engines, err := factory.New(ctx, packages, &iextengine.ExtEngineConfig{}, 1)
	if err != nil {
		panic(err)
	}
	extEngine := engines[0]
	defer extEngine.Close(ctx)
	//
	// Invoke command
	//
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, newOrderCmd.Entity()), cmdProcState)
	require.NoError(err)
	//
	// Invoke projector
	//
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, calcOrderedItemsProjector.Entity()), actualizerState)
	require.NoError(err)
	ready, err := actualizerState.ApplyIntents()
	require.NoError(err)
	require.False(ready)

	// Invoke projector again
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, calcOrderedItemsProjector.Entity()), actualizerState))
	ready, err = actualizerState.ApplyIntents()
	require.NoError(err)
	require.False(ready)

	// Flush bundles with intents
	require.NoError(actualizerState.FlushBundles())

	// Test view, must be calculated from 2 events
	kb := app.ViewRecords().KeyBuilder(orderedItemsView)
	kb.PartitionKey().PutInt32("Year", 2023)
	kb.ClusteringColumns().PutInt32("Month", 1)
	kb.ClusteringColumns().PutInt32("Day", 1)
	value, err := app.ViewRecords().Get(ws, kb)

	require.NoError(err)
	require.NotNil(value)
	require.Equal(int64(400), value.AsInt64("Amount"))
}

func appStructs(appDef appdef.IAppDefBuilder, prepareAppCfg appCfgCallback) istructs.IAppStructs {
	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddConfig(istructs.AppQName_test1_app1, appDef)
	if prepareAppCfg != nil {
		prepareAppCfg(cfg)
	}

	asf := mem.Provide()
	storageProvider := istorageimpl.Provide(asf)
	prov := istructsmem.Provide(
		cfgs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		storageProvider)
	structs, err := prov.AppStructs(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}
	return structs
}

func appStructsFromCallback(prepareAppDef appDefCallback, prepareAppCfg appCfgCallback) istructs.IAppStructs {
	appDef := appdef.New()
	if prepareAppDef != nil {
		prepareAppDef(appDef)
	}
	return appStructs(appDef, prepareAppCfg)
}

func requireMemStat(t *testing.T, wasmEngine *wazeroExtEngine, mallocs, frees, heapInUse uint32) {
	m, err := wasmEngine.getMallocs(testPkg, context.Background())
	require.NoError(t, err)
	f, err := wasmEngine.getFrees(testPkg, context.Background())
	require.NoError(t, err)
	h, err := wasmEngine.getHeapinuse(testPkg, context.Background())
	require.NoError(t, err)

	require.Equal(t, mallocs, uint32(m))
	require.Equal(t, frees, uint32(f))
	require.Equal(t, heapInUse, uint32(h))
}

func requireMemStatEx(t *testing.T, wasmEngine *wazeroExtEngine, mallocs, frees, heapSys, heapInUse uint32) {
	requireMemStat(t, wasmEngine, mallocs, frees, heapInUse)
	h, err := wasmEngine.getHeapSys(testPkg, context.Background())
	require.NoError(t, err)
	require.Equal(t, heapSys, uint32(h))
}

func testFactoryHelper(ctx context.Context, moduleUrl *url.URL, funcs []string, cfg iextengine.ExtEngineConfig, compile bool) (iextengine.IExtensionEngine, error) {
	packages := []iextengine.ExtensionPackage{
		{
			QualifiedName:  testPkg,
			ModuleUrl:      moduleUrl,
			ExtensionNames: funcs,
		},
	}
	engines, err := ProvideExtensionEngineFactory(compile).New(ctx, packages, &cfg, 1)
	if err != nil {
		return nil, err
	}
	return engines[0], nil
}

func Test_Allocs_ManualGC(t *testing.T) {

	const arrAppend = "arrAppend"
	const arrReset = "arrReset"

	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/allocs/pkggc.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleUrl, []string{arrAppend, arrReset}, iextengine.ExtEngineConfig{}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)

	wasmEngine := extEngine.(*wazeroExtEngine)

	const expectedHeapSize = uint32(1999568)

	requireMemStatEx(t, wasmEngine, 1, 0, expectedHeapSize, WasmPreallocatedBufferSize)

	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend), extIO))
	requireMemStatEx(t, wasmEngine, 3, 0, expectedHeapSize, WasmPreallocatedBufferSize+32)

	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend), extIO))
	requireMemStatEx(t, wasmEngine, 5, 0, expectedHeapSize, WasmPreallocatedBufferSize+64)

	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend), extIO))
	requireMemStatEx(t, wasmEngine, 7, 0, expectedHeapSize, WasmPreallocatedBufferSize+6*16)

	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrReset), extIO))
	requireMemStatEx(t, wasmEngine, 7, 0, expectedHeapSize, WasmPreallocatedBufferSize+6*16)

	require.NoError(wasmEngine.gc(testPkg, ctx))
	requireMemStatEx(t, wasmEngine, 7, 6, expectedHeapSize, WasmPreallocatedBufferSize)
}

func Test_Allocs_AutoGC(t *testing.T) {

	const arrAppend = "arrAppend"
	const arrReset = "arrReset"

	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/allocs/pkggc.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleUrl, []string{arrAppend, arrReset}, iextengine.ExtEngineConfig{MemoryLimitPages: 0xffff}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)

	const expectedHeapSize = uint32(1999568)
	var expectedAllocs = uint32(1)
	var expectedFrees = uint32(0)
	wasmEngine := extEngine.(*wazeroExtEngine)

	requireMemStatEx(t, wasmEngine, expectedAllocs, expectedFrees, expectedHeapSize, WasmPreallocatedBufferSize)

	calculatedHeapInUse := WasmPreallocatedBufferSize
	for calculatedHeapInUse < expectedHeapSize-16 {
		require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend), extIO))
		require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrReset), extIO))
		calculatedHeapInUse += 32
		expectedAllocs += 2
	}

	// no GC has been called, and the heap is now full
	requireMemStatEx(t, wasmEngine, expectedAllocs, expectedFrees, expectedHeapSize, expectedHeapSize-16)

	// next call will trigger auto-GC
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend), extIO))
	expectedAllocs += 2
	expectedFrees += expectedAllocs - 7
	requireMemStat(t, wasmEngine, expectedAllocs, expectedFrees, WasmPreallocatedBufferSize+96)

	// next call will not trigger auto-GC
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrReset), extIO))
	requireMemStat(t, wasmEngine, expectedAllocs, expectedFrees, WasmPreallocatedBufferSize+96) // stays the same

}

func Test_NoGc_MemoryOverflow(t *testing.T) {

	const arrAppend = "arrAppend"
	const arrReset = "arrReset"

	require := require.New(t)
	ctx := context.Background()

	moduleUrl := testModuleURL("./_testdata/allocs/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleUrl, []string{}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x10}, false)
	require.ErrorContains(err, "the minimum limit of memory is: 1700000.0 bytes, requested limit is: 1048576.0")
	require.Nil(extEngine)

	extEngine, err = testFactoryHelper(ctx, moduleUrl, []string{arrAppend, arrReset}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)

	wasmEngine := extEngine.(*wazeroExtEngine)

	var expectedAllocs = uint32(1)
	var expectedFrees = uint32(0)

	requireMemStatEx(t, wasmEngine, expectedAllocs, expectedFrees, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)

	calculatedHeapInUse := WasmPreallocatedBufferSize
	err = nil
	for calculatedHeapInUse < 0x20*iextengine.MemoryPageSize {
		err = wasmEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend), extIO)
		if err != nil {
			break
		}
		err = wasmEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrReset), extIO)
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
	extEngine, err := testFactoryHelper(ctx, moduleUrl, []string{"longFunc"}, iextengine.ExtEngineConfig{}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)

	maxDuration := time.Millisecond * 50
	ctxTimeout, cancel := context.WithTimeout(context.Background(), maxDuration)
	defer cancel()

	t0 := time.Now()
	err = extEngine.Invoke(ctxTimeout, appdef.NewFullQName(testPkg, "longFunc"), extIO)

	require.ErrorIs(err, sys.NewExitError(sys.ExitCodeDeadlineExceeded))
	require.Less(time.Since(t0), maxDuration*4)
}

type panicsUnit struct {
	name   string
	expect string
}

func Test_HandlePanics(t *testing.T) {

	tests := []panicsUnit{
		{"incorrectStorageQname", "invalid string representation of qualified name: foo"},
		{"incorrectEntityQname", "invalid string representation of qualified name: abc"},
		{"unsupportedStorage", "unsupported storage"},
		{"incorrectKeyBuilder", PanicIncorrectKeyBuilder},
		{"canExistIncorrectKey", PanicIncorrectKeyBuilder},
		{"mustExistIncorrectKey", PanicIncorrectKeyBuilder},
		{"readIncorrectKeyBuilder", PanicIncorrectKeyBuilder},
		{"incorrectKey", PanicIncorrectKey},
		{"incorrectValue", PanicIncorrectValue},
		{"incorrectValue2", PanicIncorrectValue},
		{"incorrectValue3", PanicIncorrectValue},
		{"mustExist", state.ErrNotExists.Error()},
		{"incorrectKeyBuilderOnNewValue", PanicIncorrectKeyBuilder},
		{"incorrectKeyBuilderOnUpdateValue", PanicIncorrectKeyBuilder},
		{"incorrectValueOnUpdateValue", PanicIncorrectValue},
		{"incorrectIntentId", PanicIncorrectIntent},
		{"readPanic", PanicIncorrectValue},
		{"readError", errTestIOError.Error()},
		{"queryError", errTestIOError.Error()},
		{"newValueError", errTestIOError.Error()},
		{"updateValueError", errTestIOError.Error()},
		{"asStringMemoryOverflow", "alloc"},
	}

	extNames := make([]string, 0, len(tests))
	for _, test := range tests {
		extNames = append(extNames, test.name)
	}

	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/panics/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleUrl, extNames, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)

	for _, test := range tests {
		e := extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, test.name), extIO)
		require.ErrorContains(e, test.expect)
	}
}

func Test_QueryValue(t *testing.T) {
	const testQueryValue = "testQueryValue"

	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/tests/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleUrl, []string{testQueryValue}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)

	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, testQueryValue), extIO)
	require.NoError(err)
}

func Test_RecoverEngine(t *testing.T) {

	const arrAppend2 = "arrAppend2"
	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/allocs/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleUrl, []string{arrAppend2}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)

	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.Error(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))

	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.Error(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
}

func Test_Read(t *testing.T) {
	const testRead = "testRead"
	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/tests/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleUrl, []string{testRead}, iextengine.ExtEngineConfig{}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, testRead), extIO)
	require.NoError(err)
}

func Test_AsBytes(t *testing.T) {
	const asBytes = "asBytes"
	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/tests/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleUrl, []string{asBytes}, iextengine.ExtEngineConfig{}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)
	wasmEngine := extEngine.(*wazeroExtEngine)
	requireMemStatEx(t, wasmEngine, 1, 0, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, asBytes), extIO)
	require.NoError(err)
	requireMemStatEx(t, wasmEngine, 2, 0, WasmPreallocatedBufferSize+2000000, WasmPreallocatedBufferSize+2000000)
}

func Test_AsBytesOverflow(t *testing.T) {
	const asBytes = "asBytes"
	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/tests/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleUrl, []string{asBytes}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)
	wasmEngine := extEngine.(*wazeroExtEngine)
	requireMemStatEx(t, wasmEngine, 1, 0, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, asBytes), extIO)
	require.ErrorContains(err, "alloc")
}

func Test_KeyPutQName(t *testing.T) {
	const putQName = "keyPutQName"
	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/tests/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleUrl, []string{putQName}, iextengine.ExtEngineConfig{}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)
	wasmEngine := extEngine.(*wazeroExtEngine)
	requireMemStatEx(t, wasmEngine, 1, 0, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, putQName), extIO)
	require.NoError(err)
}

func Test_NoAllocs(t *testing.T) {
	const testNoAllocs = "testNoAllocs"
	extIO = &mockIo{}
	projectorMode = false
	require := require.New(t)
	ctx := context.Background()
	moduleUrl := testModuleURL("./_testdata/tests/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleUrl, []string{testNoAllocs}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)

	wasmEngine := extEngine.(*wazeroExtEngine)
	requireMemStatEx(t, wasmEngine, 1, 0, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)

	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, testNoAllocs), extIO)
	require.NoError(err)

	requireMemStatEx(t, wasmEngine, 1, 0, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)
	require.Len(extIO.intents, 2)
	v0 := extIO.intents[0].value.(*mockValueBuilder)

	// new value
	require.Equal("test@gmail.com", v0.items["from"])
	require.Equal(int32(668), v0.items["port"])
	require.Equal(appdef.NewQName(testPackageLocalPath, "test"), v0.items["qname"])
	bytes := (v0.items["key"]).([]byte)
	require.Len(bytes, 5)

	v1 := extIO.intents[1].value.(*mockValueBuilder)
	require.Equal(int32(12346), v1.items["offs"])
	require.Equal("sys.InvitationAccepted", v1.items["qname"])
}

func Test_WithState(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()

	testView := appdef.NewQName("pkg", "TestView")
	const extension = "incrementProjector"
	const cc = "cc"
	const pk = "pk"
	const vv = "vv"
	const intentsLimit = 5
	const bundlesLimit = 5
	const ws = istructs.WSID(1)

	// build app
	app := appStructsFromCallback(
		func(appDef appdef.IAppDefBuilder) {
			appDef.AddPackage("pkg", "pkg")
			projectors.ProvideViewDef(appDef, testView, func(view appdef.IViewBuilder) {
				view.Key().PartKey().AddField(pk, appdef.DataKind_int32)
				view.Key().ClustCols().AddField(cc, appdef.DataKind_int32)
				view.Value().AddField(vv, appdef.DataKind_int32, true)
			})
		},
		func(cfg *istructsmem.AppConfigType) {})
	state := state.ProvideAsyncActualizerStateFactory()(context.Background(), app, nil, state.SimpleWSIDFunc(ws), nil, nil, nil, intentsLimit, bundlesLimit)

	// build packages
	moduleUrl := testModuleURL("./_testdata/basicusage/pkg.wasm")
	packages := []iextengine.ExtensionPackage{
		{
			QualifiedName:  testPkg,
			ModuleUrl:      moduleUrl,
			ExtensionNames: []string{extension},
		},
	}

	// build extension engine
	factory := ProvideExtensionEngineFactory(true)
	engines, err := factory.New(ctx, packages, &iextengine.ExtEngineConfig{}, 1)
	if err != nil {
		panic(err)
	}
	extEngine := engines[0]
	defer extEngine.Close(ctx)

	// Invoke extension
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, extension), state))
	ready, err := state.ApplyIntents()
	require.NoError(err)
	require.False(ready)

	// Invoke extension again
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, extension), state))
	ready, err = state.ApplyIntents()
	require.NoError(err)
	require.False(ready)

	// Flush bundles with intents
	require.NoError(state.FlushBundles())

	// Test view
	kb := app.ViewRecords().KeyBuilder(testView)
	kb.PartitionKey().PutInt32(pk, 1)
	kb.ClusteringColumns().PutInt32(cc, 1)
	value, err := app.ViewRecords().Get(ws, kb)

	require.NoError(err)
	require.NotNil(value)
	require.Equal(int32(2), value.AsInt32(vv))
}

func Test_StatePanic(t *testing.T) {

	testView := appdef.NewQName("pkg", "TestView")
	const cc = "cc"
	const pk = "pk"
	const vv = "vv"
	const intentsLimit = 5
	const bundlesLimit = 5
	const ws = istructs.WSID(1)

	app := appStructsFromCallback(
		func(appDef appdef.IAppDefBuilder) {
			appDef.AddPackage("pkg", "pkg")
			projectors.ProvideViewDef(appDef, testView, func(view appdef.IViewBuilder) {
				view.Key().PartKey().AddField(pk, appdef.DataKind_int32)
				view.Key().ClustCols().AddField(cc, appdef.DataKind_int32)
				view.Value().AddField(vv, appdef.DataKind_int32, true)
			})
		},
		func(cfg *istructsmem.AppConfigType) {})
	state := state.ProvideAsyncActualizerStateFactory()(context.Background(), app, nil, state.SimpleWSIDFunc(ws), nil, nil, nil, intentsLimit, bundlesLimit)

	const extname = "wrongFieldName"
	const undefinedPackage = "undefinedPackage"

	require := require.New(t)
	ctx := context.Background()

	moduleUrl := testModuleURL("./_testdata/panics/pkg.wasm")
	packages := []iextengine.ExtensionPackage{
		{
			QualifiedName:  testPkg,
			ModuleUrl:      moduleUrl,
			ExtensionNames: []string{extname, undefinedPackage},
		},
	}
	factory := ProvideExtensionEngineFactory(true)
	engines, err := factory.New(ctx, packages, &iextengine.ExtEngineConfig{}, 1)
	if err != nil {
		panic(err)
	}
	extEngine := engines[0]
	defer extEngine.Close(ctx)
	//
	// Invoke extension
	//
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, extname), state)
	require.ErrorContains(err, "int32-type field «wrong» is not found")

	//
	// Invoke extension
	//
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, undefinedPackage), state)
	require.ErrorContains(err, errUndefinedPackage("github.com/company/pkg").Error())
}

type (
	appDefCallback func(appDef appdef.IAppDefBuilder)
	appCfgCallback func(cfg *istructsmem.AppConfigType)
)

//go:embed sql_example_syspkg/*.sql
var sfs embed.FS

func appStructsFromSQL(packagePath string, appdefSql string, prepareAppCfg appCfgCallback) istructs.IAppStructs {
	appDef := appdef.New()

	fs, err := parser.ParseFile("file1.sql", appdefSql)
	if err != nil {
		panic(err)
	}

	pkg, err := parser.BuildPackageSchema(packagePath, []*parser.FileSchemaAST{fs})
	if err != nil {
		panic(err)
	}

	pkgSys, err := parser.ParsePackageDir(appdef.SysPackage, sfs, "sql_example_syspkg")
	if err != nil {
		panic(err)
	}

	packages, err := parser.BuildAppSchema([]*parser.PackageSchemaAST{
		pkgSys,
		pkg,
	})
	if err != nil {
		panic(err)
	}

	err = parser.BuildAppDefs(packages, appDef)
	if err != nil {
		panic(err)
	}

	return appStructs(appDef, prepareAppCfg)
}
