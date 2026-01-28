/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

package iextenginewazero

import (
	"context"
	"embed"
	"fmt"
	"math"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/tetratelabs/wazero/sys"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/isequencer"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/state/safestate"
	"github.com/voedger/voedger/pkg/state/stateprovide"
	"github.com/voedger/voedger/pkg/sys/authnz"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	"github.com/voedger/voedger/pkg/state"
)

var extIO = &mockIo{}
var plogOffset istructs.Offset
var wlogOffset istructs.Offset
var newWorkspaceCmd = appdef.NewQName("sys", "NewWorkspace")
var testView = appdef.NewQName(testPkg, "TestView")
var dummyCommand = appdef.NewQName(testPkg, "Dummy")
var dummyProj = appdef.NewQName(testPkg, "DummyProj")
var testWorkspaceDescriptor = appdef.NewQName(testPkg, "RestaurantDescriptor")
var testVVMName = processors.VVMName("test VVM name")
var testApp = istructs.AppQName_test1_app1

const testPkg = "mypkg"
const ws = istructs.WSID(1)
const partition = istructs.PartitionID(1)

func Test_BasicUsage(t *testing.T) {
	// Test Consts
	const intentsLimit = 5
	const bundlesLimit = 5
	require := require.New(t)
	newOrderCmd := appdef.NewQName(testPkg, "NewOrder")
	calcOrderedItemsProjector := appdef.NewQName(testPkg, "CalcOrderedItems")
	orderedItemsView := appdef.NewQName(testPkg, "OrderedItems")

	// Prepare app
	app := appStructsFromSQL("github.com/untillpro/airs-bp3/packages/"+testPkg, `APPLICATION test();
		WORKSPACE Restaurant (
			DESCRIPTOR RestaurantDescriptor ();
			TABLE Order INHERITS sys.ODoc (
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
				PROJECTOR CalcOrderedItems AFTER EXECUTE ON NewOrder INTENTS(sys.View(OrderedItems));
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
			PLogOffset:        plogOffset + 1,
			QName:             newOrderCmd,
			WLogOffset:        wlogOffset + 1,
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
	cmdPrepareArgsFunc := func() istructs.CommandPrepareArgs {
		return istructs.CommandPrepareArgs{
			PrepareArgs: istructs.PrepareArgs{
				Workpiece:      nil,
				ArgumentObject: event.ArgumentObject(),
				WSID:           ws,
				Workspace:      nil,
			},
			ArgumentUnloggedObject: nil,
		}
	}
	argFunc := func() istructs.IObject { return event.ArgumentObject() }
	unloggedArgFunc := func() istructs.IObject { return nil }
	appFunc := func() istructs.IAppStructs { return app }
	wlogOffsetFunc := func() istructs.Offset { return event.WLogOffset() }
	originFunc := func() string { return "" }

	// Create states for Command processor and Actualizer
	actualizerState := stateprovide.ProvideAsyncActualizerStateFactory()(context.Background(), appFunc, nil, state.SimpleWSIDFunc(ws), nil, nil, eventFunc, nil, nil, intentsLimit, bundlesLimit, state.NullOpts, nil)
	cmdProcState := stateprovide.ProvideCommandProcessorStateFactory()(context.Background(), appFunc, nil, state.SimpleWSIDFunc(ws), nil, cudFunc, nil, nil, intentsLimit, nil, cmdPrepareArgsFunc, argFunc, unloggedArgFunc, wlogOffsetFunc, state.NullOpts, originFunc)

	// Create extension package from WASM
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/basicusage/pkg.wasm")
	packages := []iextengine.ExtensionModule{
		{
			Path:           testPkg,
			ModuleURL:      moduleURL,
			ExtensionNames: []string{calcOrderedItemsProjector.Entity(), newOrderCmd.Entity()},
		},
	}

	// Create extension engine
	metrics := imetrics.Provide()
	factory := ProvideExtensionEngineFactory(iextengine.WASMFactoryConfig{Compile: true}, testVVMName, metrics)
	engines, err := factory.New(ctx, app.AppQName(), packages, &iextengine.ExtEngineConfig{}, 1)
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

	testMetrics(require, metrics, expectedMetrics{
		errors:           0,
		invocationsTotal: 3,
		recovers:         0,
	})

}

type expectedMetrics struct {
	errors           int
	invocationsTotal int
	recovers         int
}

func testMetrics(require *require.Require, metrics imetrics.IMetrics, expectedMetrcis expectedMetrics) {
	checkedMetricsCount := 0
	err := metrics.List(func(metric imetrics.IMetric, metricValue float64) (err error) {
		switch metric.Name() {
		case metric_voedger_pee_invocations_total:
			require.EqualValues(expectedMetrcis.invocationsTotal, metricValue)
			checkedMetricsCount++
		case metric_voedger_pee_errors_total:
			require.EqualValues(expectedMetrcis.errors, metricValue)
			checkedMetricsCount++
		case metric_voedger_pee_invocations_seconds:
			require.GreaterOrEqual(metricValue, float64(0))
			checkedMetricsCount++
		case metric_voedger_pee_recovers_total:
			require.EqualValues(expectedMetrcis.recovers, metricValue)
			checkedMetricsCount++
		default:
			panic("unexpected metric: " + metric.Name())
		}
		return nil
	})
	require.NoError(err)
	require.Equal(4, checkedMetricsCount)
}

func appStructs(appDef appdef.IAppDefBuilder, prepareAppCfg appCfgCallback) istructs.IAppStructs {
	cfgs := make(istructsmem.AppConfigsType, 1)
	cfg := cfgs.AddBuiltInAppConfig(istructs.AppQName_test1_app1, appDef)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	if prepareAppCfg != nil {
		prepareAppCfg(cfg)
		cfg.Resources.Add(istructsmem.NewCommandFunction(newWorkspaceCmd, istructsmem.NullCommandExec))
	}

	asf := mem.Provide(testingu.MockTime)
	storageProvider := istorageimpl.Provide(asf)
	prov := istructsmem.Provide(
		cfgs,
		iratesce.TestBucketsFactory,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()),
		storageProvider,
		isequencer.SequencesTrustLevel_0, nil)
	structs, err := prov.BuiltIn(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}
	return structs
}

func requireMemStat(t *testing.T, wasmEngine *wazeroExtEngine, mallocs, frees, heapInUse uint32) {
	require := require.New(t)

	m, err := wasmEngine.getMallocs(testPkg, context.Background())
	require.NoError(err)
	f, err := wasmEngine.getFrees(testPkg, context.Background())
	require.NoError(err)
	h, err := wasmEngine.getHeapinuse(testPkg, context.Background())
	require.NoError(err)

	require.Equal(mallocs, uint32(m))
	require.Equal(frees, uint32(f))
	require.Equal(heapInUse, uint32(h))
}

func requireMemStatEx(t *testing.T, wasmEngine *wazeroExtEngine, mallocs, frees, heapSys, heapInUse uint32) {
	require := require.New(t)

	requireMemStat(t, wasmEngine, mallocs, frees, heapInUse)
	h, err := wasmEngine.getHeapSys(testPkg, context.Background())
	require.NoError(err)
	require.Equal(heapSys, uint32(h))
}

func testFactoryHelperWithMetrics(ctx context.Context, moduleURL *url.URL, funcs []string, cfg iextengine.ExtEngineConfig, compile bool) (iextengine.IExtensionEngine, imetrics.IMetrics, error) {
	packages := []iextengine.ExtensionModule{
		{
			Path:           testPkg,
			ModuleURL:      moduleURL,
			ExtensionNames: funcs,
		},
	}
	imetrics := imetrics.Provide()
	engines, err := ProvideExtensionEngineFactory(iextengine.WASMFactoryConfig{Compile: compile}, testVVMName, imetrics).New(ctx, testApp, packages, &cfg, 1)
	if err != nil {
		return nil, nil, err
	}
	return engines[0], imetrics, nil
}

func testFactoryHelper(ctx context.Context, moduleURL *url.URL, funcs []string, cfg iextengine.ExtEngineConfig, compile bool) (iextengine.IExtensionEngine, error) {
	extEngine, _, err := testFactoryHelperWithMetrics(ctx, moduleURL, funcs, cfg, compile)
	return extEngine, err
}

func Test_Allocs_ManualGC(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	const arrAppend = "arrAppend"
	const arrReset = "arrReset"

	require := require.New(t)
	ctx := context.Background()
	WasmPreallocatedBufferSize = 1000000
	moduleURL := testModuleURL("./_testdata/allocs/pkggc.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{arrAppend, arrReset}, iextengine.ExtEngineConfig{}, false)
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
	if testing.Short() {
		t.Skip()
	}

	const arrAppend = "arrAppend"
	const arrReset = "arrReset"
	WasmPreallocatedBufferSize = 1000000
	require := require.New(t)
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/allocs/pkggc.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{arrAppend, arrReset}, iextengine.ExtEngineConfig{MemoryLimitPages: 0xffff}, false)
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
	if testing.Short() {
		t.Skip()
	}

	const arrAppend = "arrAppend"
	const arrReset = "arrReset"
	WasmPreallocatedBufferSize = 1000000

	require := require.New(t)
	ctx := context.Background()

	moduleURL := testModuleURL("./_testdata/allocs/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x10}, false)
	require.ErrorContains(err, "the minimum limit of memory is: 1700000.0 bytes, requested limit is: 1048576.0")
	require.Nil(extEngine)

	extEngine, err = testFactoryHelper(ctx, moduleURL, []string{arrAppend, arrReset}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)

	wasmEngine := extEngine.(*wazeroExtEngine)
	wasmEngine.autoRecover = false

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
	require.EqualError(err, "panic: runtime error: out of memory")
}

func Test_SetLimitsExecutionInterval(t *testing.T) {

	require := require.New(t)
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/allocs/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{"longFunc"}, iextengine.ExtEngineConfig{}, false)
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

	WasmPreallocatedBufferSize = 1000000
	tests := []panicsUnit{
		{"incorrectStorageQname", "convert error: string «foo»"},
		{"incorrectEntityQname", "convert error: string «abc»"},
		{"unsupportedStorage", "unsupported storage"},
		{"incorrectKeyBuilder", safestate.PanicIncorrectKeyBuilder},
		{"canExistIncorrectKey", safestate.PanicIncorrectKeyBuilder},
		{"mustExistIncorrectKey", safestate.PanicIncorrectKeyBuilder},
		{"readIncorrectKeyBuilder", safestate.PanicIncorrectKeyBuilder},
		{"incorrectKey", safestate.PanicIncorrectKey},
		{"incorrectValue", safestate.PanicIncorrectValue},
		{"incorrectValue2", safestate.PanicIncorrectValue},
		{"incorrectValue3", safestate.PanicIncorrectValue},
		{"mustExist", stateprovide.ErrNotExists.Error()},
		{"incorrectKeyBuilderOnNewValue", safestate.PanicIncorrectKeyBuilder},
		{"incorrectKeyBuilderOnUpdateValue", safestate.PanicIncorrectKeyBuilder},
		{"incorrectValueOnUpdateValue", safestate.PanicIncorrectValue},
		{"incorrectIntentId", safestate.PanicIncorrectIntent},
		{"readPanic", safestate.PanicIncorrectValue},
		{"readError", errTestIOError.Error()},
		{"queryError", errTestIOError.Error()},
		{"newValueError", errTestIOError.Error()},
		{"updateValueError", errTestIOError.Error()},
		{"asStringMemoryOverflow", "runtime error: out of memory"},
	}

	extNames := make([]string, 0, len(tests))
	for _, test := range tests {
		extNames = append(extNames, test.name)
	}

	require := require.New(t)
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/panics/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, extNames, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)

	for _, test := range tests {
		e := extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, test.name), extIO)
		require.ErrorContains(e, test.expect)
	}
}

func Test_ResetStdout(t *testing.T) {
	const testResetStdout = "TestPanic"
	require := require.New(t)
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/panics/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{testResetStdout}, iextengine.ExtEngineConfig{}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, testResetStdout), extIO)
	require.ErrorContains(err, "world")
	error1 := err.Error()
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, testResetStdout), extIO)
	require.ErrorContains(err, "world")
	require.Equal(error1, err.Error())
}

func Test_QueryValue(t *testing.T) {
	const testQueryValue = "testQueryValue"

	require := require.New(t)
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/tests/wasm/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{testQueryValue}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)

	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, testQueryValue), extIO)
	require.NoError(err)
}

func mvf(m *imetrics.MetricValue) float64 {
	ptr := (*uint64)(unsafe.Pointer(m))
	return math.Float64frombits(atomic.LoadUint64(ptr))
}

func mv(m *imetrics.MetricValue) int {
	return int(mvf(m))
}

func Test_RecoverEngine(t *testing.T) {

	testWithPreallocatedBuffer := func(t *testing.T, preallocatedBufferSize uint32) {
		t.Run(fmt.Sprintf("PreallocatedBufferSize=%d", preallocatedBufferSize), func(t *testing.T) {
			WasmPreallocatedBufferSize = preallocatedBufferSize
			const arrAppend2 = "arrAppend2"
			require := require.New(t)
			ctx := context.Background()
			moduleURL := testModuleURL("./_testdata/allocs/pkg.wasm")
			extEngine, metrics, err := testFactoryHelperWithMetrics(ctx, moduleURL, []string{arrAppend2}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, true)
			require.NoError(err)
			defer extEngine.Close(ctx)
			we := extEngine.(*wazeroExtEngine)

			totalRuns := 0
			totalErrors := 0

			invocationsTotal := metrics.AppMetricAddr(metric_voedger_pee_invocations_total, string(testVVMName), testApp)
			invocationsSeconds := metrics.AppMetricAddr(metric_voedger_pee_invocations_seconds, string(testVVMName), testApp)
			eTotal := metrics.AppMetricAddr(metric_voedger_pee_errors_total, string(testVVMName), testApp)
			recoversTotal := metrics.AppMetricAddr(metric_voedger_pee_recovers_total, string(testVVMName), testApp)

			require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
			totalRuns++
			require.Equal(0, mv(recoversTotal))
			require.Equal(totalRuns, mv(invocationsTotal))
			require.Equal(totalErrors, mv(eTotal))
			heapInUseAfterFirstInvoke, err := we.getHeapinuse(testPkg, context.Background())
			require.NoError(err)

			for recoverNo := 0; recoverNo < 10; recoverNo++ { // 10 recover cycles
				for run := 1; mv(recoversTotal) == recoverNo; { // run until auto-recover is triggered{
					require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
					totalRuns++
					if mv(recoversTotal) != recoverNo {
						totalRuns++ // recovered, means one more invocation
						totalErrors++
					}
					require.Equal(totalRuns, mv(invocationsTotal))
					require.Equal(totalErrors, mv(eTotal))
					run++
				}
				require.Equal(recoverNo+1, mv(recoversTotal))
				heapInUseAfterRecover, err := we.getHeapinuse(testPkg, context.Background())
				require.NoError(err)
				require.Equal(heapInUseAfterRecover, heapInUseAfterFirstInvoke)
			}

			require.Greater(mvf(invocationsSeconds), 0.0)
		})
	}

	testWithPreallocatedBuffer(t, 1000000)
	testWithPreallocatedBuffer(t, 900000)
	testWithPreallocatedBuffer(t, 800000)
	testWithPreallocatedBuffer(t, 700000)

	testWithPreallocatedBuffer(t, 600000)
	testWithPreallocatedBuffer(t, 500000)

	testWithPreallocatedBuffer(t, 200000)
	testWithPreallocatedBuffer(t, 100000)
	testWithPreallocatedBuffer(t, WasmDefaultPreallocatedBufferSize)
}

func Test_RecoverEngine2(t *testing.T) {

	WasmPreallocatedBufferSize = 1000000
	const arrAppend2 = "arrAppend2"
	require := require.New(t)
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/allocs/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{arrAppend2}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, true)
	require.NoError(err)
	defer extEngine.Close(ctx)
	we := extEngine.(*wazeroExtEngine)

	var recoversTotal imetrics.MetricValue
	we.recoversTotal = &recoversTotal
	var log []string
	logger.PrintLine = func(level logger.TLogLevel, line string) {
		log = append(log, line)
	}

	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.Equal(0, mv(&recoversTotal))
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.Equal(0, mv(&recoversTotal))
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.Equal(0, mv(&recoversTotal))

	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.Equal(1, mv(&recoversTotal))
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.Equal(1, mv(&recoversTotal))
	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.Equal(1, mv(&recoversTotal))
	require.Len(log, 1)
	require.Contains(log[0], "test1/app1/mypkg wazero engine recovered")

	require.NoError(extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, arrAppend2), extIO))
	require.Equal(2, mv(&recoversTotal))
	require.Len(log, 2)
	require.Contains(log[1], "test1/app1/mypkg wazero engine recovered")
}

func Test_Read(t *testing.T) {
	const testRead = "testRead"
	require := require.New(t)
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/tests/wasm/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{testRead}, iextengine.ExtEngineConfig{}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, testRead), extIO)
	require.NoError(err)
}

func Test_AsBytes(t *testing.T) {
	const asBytes = "asBytes"
	require := require.New(t)
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/tests/wasm/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{asBytes}, iextengine.ExtEngineConfig{}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)
	wasmEngine := extEngine.(*wazeroExtEngine)
	requireMemStatEx(t, wasmEngine, 1, 0, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, asBytes), extIO)
	require.NoError(err)
	requireMemStatEx(t, wasmEngine, 2, 0, WasmPreallocatedBufferSize+2000000, WasmPreallocatedBufferSize+2000000)
}

func Test_AsBytesOverflow(t *testing.T) {
	WasmPreallocatedBufferSize = 1000000
	const asBytes = "asBytes"
	require := require.New(t)
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/tests/wasm/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{asBytes}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, false)
	require.NoError(err)
	defer extEngine.Close(ctx)
	wasmEngine := extEngine.(*wazeroExtEngine)
	requireMemStatEx(t, wasmEngine, 1, 0, WasmPreallocatedBufferSize, WasmPreallocatedBufferSize)
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, asBytes), extIO)
	require.EqualError(err, "panic: runtime error: out of memory")
}

func Test_KeyPutQName(t *testing.T) {
	const putQName = "keyPutQName"
	require := require.New(t)
	ctx := context.Background()
	moduleURL := testModuleURL("./_testdata/tests/wasm/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{putQName}, iextengine.ExtEngineConfig{}, false)
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
	moduleURL := testModuleURL("./_testdata/tests/wasm/pkg.wasm")
	extEngine, err := testFactoryHelper(ctx, moduleURL, []string{testNoAllocs}, iextengine.ExtEngineConfig{MemoryLimitPages: 0x20}, false)
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

	const extension = "incrementProjector"
	const cc = "cc"
	const pk = "pk"
	const vv = "vv"
	const intentsLimit = 5
	const bundlesLimit = 5
	const ws = istructs.WSID(1)

	app := appStructsFromSQL(testPkg, `APPLICATION test();
		WORKSPACE Restaurant (
			DESCRIPTOR RestaurantDescriptor ();
			VIEW TestView (
				pk int32,
				cc int32,
				vv int32,
				PRIMARY KEY ((pk), cc)
			) AS RESULT OF DummyProj;
			EXTENSION ENGINE WASM(
				COMMAND Dummy();
				PROJECTOR DummyProj AFTER EXECUTE ON Dummy INTENTS(sys.View(TestView));
			);
		)`,
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(dummyCommand, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(istructs.Projector{Name: dummyProj})
		})

	// build app
	appFunc := func() istructs.IAppStructs { return app }
	state := stateprovide.ProvideAsyncActualizerStateFactory()(context.Background(), appFunc, nil, state.SimpleWSIDFunc(ws), nil, nil, nil, nil, nil, intentsLimit, bundlesLimit, state.NullOpts, nil)

	// build packages
	moduleURL := testModuleURL("./_testdata/basicusage/pkg.wasm")
	packages := []iextengine.ExtensionModule{
		{
			Path:           testPkg,
			ModuleURL:      moduleURL,
			ExtensionNames: []string{extension},
		},
	}

	// build extension engine
	imetrics := imetrics.Provide()
	factory := ProvideExtensionEngineFactory(iextengine.WASMFactoryConfig{Compile: true}, testVVMName, imetrics)
	engines, err := factory.New(ctx, app.AppQName(), packages, &iextengine.ExtEngineConfig{}, 1)
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

	const intentsLimit = 5
	const bundlesLimit = 5
	const ws = istructs.WSID(1)

	app := appStructsFromSQL(testPkg, `APPLICATION test();
		WORKSPACE Restaurant (
			DESCRIPTOR RestaurantDescriptor ();
			VIEW TestView (
				pk int32,
				cc int32,
				vv int32,
				PRIMARY KEY ((pk), cc)
			) AS RESULT OF DummyProj;
			EXTENSION ENGINE WASM(
				COMMAND Dummy();
				PROJECTOR DummyProj AFTER EXECUTE ON Dummy INTENTS(sys.View(TestView));
			);
		)`,
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(dummyCommand, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(istructs.Projector{Name: dummyProj})
		})
	appFunc := func() istructs.IAppStructs { return app }
	state := stateprovide.ProvideAsyncActualizerStateFactory()(context.Background(), appFunc, nil, state.SimpleWSIDFunc(ws), nil, nil, nil, nil, nil, intentsLimit, bundlesLimit, state.NullOpts, nil)

	const extname = "wrongFieldName"
	const undefinedPackage = "undefinedPackage"

	require := require.New(t)
	ctx := context.Background()

	moduleURL := testModuleURL("./_testdata/panics/pkg.wasm")
	packages := []iextengine.ExtensionModule{
		{
			Path:           testPkg,
			ModuleURL:      moduleURL,
			ExtensionNames: []string{extname, undefinedPackage},
		},
	}
	imetrics := imetrics.Provide()
	factory := ProvideExtensionEngineFactory(iextengine.WASMFactoryConfig{Compile: true}, testVVMName, imetrics)
	engines, err := factory.New(ctx, app.AppQName(), packages, &iextengine.ExtEngineConfig{}, 1)
	if err != nil {
		panic(err)
	}
	extEngine := engines[0]
	defer extEngine.Close(ctx)
	//
	// Invoke extension
	//
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, extname), state)
	require.Error(err, require.Is(istructsmem.ErrNameNotFoundError), require.HasAll("TestView", "wrong"))

	//
	// Invoke extension
	//
	err = extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, undefinedPackage), state)
	require.ErrorContains(err, errUndefinedPackage("github.com/company/pkg").Error())
}

type (
	appCfgCallback func(cfg *istructsmem.AppConfigType)
)

//go:embed sql_example_syspkg/*.vsql
var sfs embed.FS

func appStructsFromSQL(packagePath string, appdefSQL string, prepareAppCfg appCfgCallback) istructs.IAppStructs {
	plogOffset = istructs.Offset(123)
	wlogOffset = istructs.Offset(42)
	appDef := builder.New()

	fs, err := parser.ParseFile("file1.vsql", appdefSQL)
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

	app := appStructs(appDef, prepareAppCfg)

	// Create workspace
	rebWs := app.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			Workspace:         ws,
			HandlingPartition: partition,
			PLogOffset:        plogOffset,
			QName:             newWorkspaceCmd,
			WLogOffset:        wlogOffset,
		},
	})
	cud := rebWs.CUDBuilder().Create(authnz.QNameCDocWorkspaceDescriptor)
	cud.PutRecordID(appdef.SystemField_ID, 1)
	cud.PutQName("WSKind", testWorkspaceDescriptor)
	rawWsEvent, err := rebWs.BuildRawEvent()
	if err != nil {
		panic(err)
	}
	wsEvent, err := app.Events().PutPlog(rawWsEvent, nil, istructsmem.NewIDGenerator())
	if err != nil {
		panic(err)
	}
	if err = app.Records().Apply(wsEvent); err != nil {
		panic(err)
	}

	return app

}

func Test_Panics(t *testing.T) {
	require := require.New(t)
	invoke := func(name string) (imetrics.IMetrics, error) {
		ctx := context.Background()
		WasmPreallocatedBufferSize = 1000000
		moduleURL := testModuleURL("./_testdata/panics/pkg.wasm")
		extEngine, metrics, err := testFactoryHelperWithMetrics(ctx, moduleURL, []string{name}, iextengine.ExtEngineConfig{}, false)
		require.NoError(err)
		defer extEngine.Close(ctx)
		return metrics, extEngine.Invoke(context.Background(), appdef.NewFullQName(testPkg, name), extIO)
	}

	t.Run("Test Panic", func(t *testing.T) {
		metrics, err := invoke("TestPanic")
		require.EqualError(err, "panic: goodbye, world")
		testMetrics(require, metrics, expectedMetrics{
			errors:           1,
			invocationsTotal: 1,
			recovers:         0,
		})
	})

	t.Run("No Panic On Sign Extensions Funcs", func(t *testing.T) {
		metrics, err := invoke("TestSignExtensionsFuncs")
		require.NoError(err)
		testMetrics(require, metrics, expectedMetrics{
			errors:           0,
			invocationsTotal: 1,
			recovers:         0,
		})
	})
}
