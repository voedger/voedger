/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package actualizers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	retrier "github.com/voedger/voedger/pkg/goutils/retry"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/sys"
)

// Design: Projection Actualizers
// https://dev.heeus.io/launchpad/#!12850
//
// Test description:
//
// 1. Initializes PLog and PLogReader, used by the
// async actualizers to read PLog
//
// 2. Stores the "saved offset" for the "decrementor"
// actualizer. Means that it will not start from the
// beginning of the PLog when launched
//
// 3. Launches async actualizers for "incrementor" and
// "decrementor" projectors.
//
// 4. Waits until both actualizers reach the end of PLog
// (projection versions updated to the most recent offset)
//
// 5. Checks projection values in different workspaces
func TestBasicUsage_AsynchronousActualizer(t *testing.T) {
	require := require.New(t)

	appName, totalPartitions, partitionNr := istructs.AppQName_test1_app1, istructs.NumAppPartitions(1), istructs.PartitionID(1) // test within partition 1

	broker, bCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                2,
		ChannelsPerSubject:      2,
		Subscriptions:           2,
		SubscriptionsPerSubject: 2,
	}, timeu.NewITime())
	defer bCleanup()

	actCfg := &BasicAsyncActualizerConfig{
		Broker: broker,
	}

	appParts, appStructs, stop := deployTestApp(
		appName, totalPartitions, false,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			ProvideViewDef(wsb, incProjectionView, buildProjectionView)
			ProvideViewDef(wsb, decProjectionView, buildProjectionView)
			wsb.AddCommand(testQName)
			wsb.AddProjector(incrementorName).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.QNames(testQName))
			wsb.AddProjector(decrementorName).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.QNames(testQName))
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(testIncrementor, testDecrementor)
		},
		actCfg)

	// store the initial actualizer offsets
	//
	// 1. there will be no stored offset for incrementor, so it starts
	// from the beginning of the log
	//
	// 2. Decrementor will have the offset=4 stored (will start from
	// 5th (index 4 in pLog array)):
	_ = storeProjectorOffset(appStructs, partitionNr, decrementorName, istructs.Offset(6))

	idGen := istructsmem.NewIDGenerator()
	createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1), idGen)
	createWS(appStructs, istructs.WSID(1002), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(2), idGen)

	f := pLogFiller{
		app:       appStructs,
		partition: partitionNr,
		offset:    istructs.Offset(3),
		cmdQName:  testQName,
	}
	f.fill(1001, idGen)
	f.fill(1002, idGen)
	f.fill(1001, idGen)
	f.fill(1001, idGen)
	f.fill(1001, idGen)
	f.fill(1002, idGen)
	f.fill(1001, idGen)
	f.fill(1001, idGen)
	f.fill(1001, idGen)
	topOffset := f.fill(1001, idGen)

	appParts.DeployAppPartitions(appName, []istructs.PartitionID{partitionNr})

	// Wait for the projectors
	for getActualizerOffset(require, appStructs, partitionNr, incrementorName) < topOffset {
		time.Sleep(time.Millisecond)
	}
	for getActualizerOffset(require, appStructs, partitionNr, decrementorName) < topOffset {
		time.Sleep(time.Millisecond)
	}

	// stop services
	stop()

	// expected projection values
	require.Equal(int32(8), getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(2), getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1002)))
	require.Equal(int32(-5), getProjectionValue(require, appStructs, decProjectionView, istructs.WSID(1001)))
	require.Equal(int32(-1), getProjectionValue(require, appStructs, decProjectionView, istructs.WSID(1002)))
}

type interruptBatchPipeline struct {
	pipeline.IAsyncPipeline
	afterFirstSend func()
	secondSendErr  error
	sends          int
}

func (p *interruptBatchPipeline) SendAsync(work pipeline.IWorkpiece) error {
	p.sends++
	if p.sends > 1 {
		return p.secondSendErr
	}
	if err := p.IAsyncPipeline.SendAsync(work); err != nil {
		return err
	}
	if p.afterFirstSend != nil {
		p.afterFirstSend()
	}
	return nil
}

func TestInterruptedPLogBatchReleasesEachEventOnce(t *testing.T) {
	require := require.New(t)
	sendErr := errors.New("send failed")
	for _, tc := range []struct {
		name            string
		cancelAfterSend bool
		expectedErr     error
	}{
		{name: "second send fails", expectedErr: sendErr},
		{name: "reader canceled after first send", cancelAfterSend: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			// Hold the queued work until the reader has released the unsent events.
			resume := make(chan struct{})
			hold := pipeline.WireAsyncFunc("hold", func(_ context.Context, work pipeline.IWorkpiece) (pipeline.IWorkpiece, error) {
				<-resume
				return work, nil
			})
			p := &interruptBatchPipeline{
				IAsyncPipeline: pipeline.NewAsyncPipeline(t.Context(), "test", hold),
				secondSendErr:  sendErr,
			}
			if tc.cancelAfterSend {
				p.afterFirstSend = cancel
			}
			drain := sync.OnceFunc(func() {
				close(resume)
				p.Close()
			})
			defer drain()

			events := []*coreutils.MockPLogEvent{{}, {}, {}}
			var batch plogBatch
			for i, event := range events {
				event.On("Release")
				batch = append(batch, plogEvent{Offset: istructs.Offset(i + 1), IPLogEvent: event})
			}
			a := asyncActualizer{pipeline: p}
			err := a.readPlogByBatches(ctx, false, func(dst *plogBatch) error {
				*dst = batch
				batch = nil
				return nil
			})
			require.ErrorIs(err, tc.expectedErr)
			events[0].AssertNumberOfCalls(t, "Release", 0)
			for _, event := range events[1:] {
				event.AssertNumberOfCalls(t, "Release", 1)
			}

			drain()
			for _, event := range events {
				event.AssertNumberOfCalls(t, "Release", 1)
			}
		})
	}
}

func TestVSQLProjectorsAsync(t *testing.T) {
	t.Run("vsql-projectors: scn: Projector runs after each successful command: sales.CreateOrder", func(t *testing.T) {
		// | command                |
		// | sales.CreateOrder      |
		// Given the application provides local command "sales.CreateOrder"
		// And the application imports command "inventory.ReserveStock"
		// And the deployed VSQL schema declares
		// When command "<command>" executes successfully
		// command = sales.CreateOrder
		all, _ := runAsyncProjectorsFeature(t, vsqlLocalCommand, "RecordCommand")
		// Then projector "RecordCommand" executes for command "<command>"
		// command = sales.CreateOrder
		require.Equal(t, int32(1), all)
	})

	t.Run("vsql-projectors: scn: Projector runs after each successful command: inventory.ReserveStock", func(t *testing.T) {
		// | command                |
		// | inventory.ReserveStock |
		// Given the application provides local command "sales.CreateOrder"
		// And the application imports command "inventory.ReserveStock"
		// And the deployed VSQL schema declares
		// When command "<command>" executes successfully
		// command = inventory.ReserveStock
		all, _ := runAsyncProjectorsFeature(t, vsqlImportedCommand, "RecordCommand")
		// Then projector "RecordCommand" executes for command "<command>"
		// command = inventory.ReserveStock
		require.Equal(t, int32(1), all)
	})

	t.Run("vsql-projectors: scn: Projector runs after each successful command: sys.CUD", func(t *testing.T) {
		// | command                |
		// | sys.CUD                |
		// Given the application provides local command "sales.CreateOrder"
		// And the application imports command "inventory.ReserveStock"
		// And the deployed VSQL schema declares
		// When command "<command>" executes successfully
		// command = sys.CUD
		all, _ := runAsyncProjectorsFeature(t, istructs.QNameCommandCUD, "RecordCommand")
		// Then projector "RecordCommand" executes for command "<command>"
		// command = sys.CUD
		require.Equal(t, int32(1), all)
	})

	t.Run("vsql-projectors: scn: Command-specific projector runs for its declared command", func(t *testing.T) {
		// Given the application provides commands "sales.CreateOrder" and "sales.CancelOrder"
		// And the deployed VSQL schema declares
		// When command "sales.CreateOrder" executes successfully
		all, specific := runAsyncProjectorsFeature(t, vsqlLocalCommand, "RecordEveryCommand")
		// Then projector "RecordEveryCommand" executes for command "sales.CreateOrder"
		require.Equal(t, int32(1), all)
		// And projector "RecordCreatedOrder" executes for command "sales.CreateOrder"
		require.Equal(t, int32(1), specific)
	})

	t.Run("vsql-projectors: scn: Command-specific projector ignores another command", func(t *testing.T) {
		// Given the application provides commands "sales.CreateOrder" and "sales.CancelOrder"
		// And the deployed VSQL schema declares
		// When command "sales.CancelOrder" executes successfully
		all, specific := runAsyncProjectorsFeature(t, vsqlCancelCommand, "RecordEveryCommand")
		// Then projector "RecordEveryCommand" executes for command "sales.CancelOrder"
		require.Equal(t, int32(1), all)
		// And projector "RecordCreatedOrder" does not execute
		require.Zero(t, specific)
	})
}

func runAsyncProjectorsFeature(t *testing.T, command appdef.QName, allProjectorEntity string) (all, specific int32) {
	t.Helper()

	var countsMu sync.Mutex
	allCounts := map[appdef.QName]int32{}
	specificCounts := map[appdef.QName]int32{}
	featureDoc := appdef.NewQName("test", "FeatureDoc")
	allProjectorName := appdef.NewQName("test", allProjectorEntity)
	specificProjectorName := appdef.NewQName("test", "RecordCreatedOrder")
	allProjector := istructs.Projector{
		Name: allProjectorName,
		Func: func(event istructs.IPLogEvent, _ istructs.IState, _ istructs.IIntents) error {
			countsMu.Lock()
			allCounts[event.QName()]++
			countsMu.Unlock()
			return nil
		},
	}
	specificProjector := istructs.Projector{
		Name: specificProjectorName,
		Func: func(event istructs.IPLogEvent, _ istructs.IState, _ istructs.IIntents) error {
			countsMu.Lock()
			specificCounts[event.QName()]++
			countsMu.Unlock()
			return nil
		},
	}

	broker, cleanupBroker := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                2,
		ChannelsPerSubject:      2,
		Subscriptions:           2,
		SubscriptionsPerSubject: 2,
	}, timeu.NewITime())
	defer cleanupBroker()
	actCfg := &BasicAsyncActualizerConfig{
		Broker: broker,
		// Must be > 0 and < FlushInterval (100ms by default),
		// so the first flush persists a skipped event's offset; 0 selects the 1m default.
		FlushPositionInterval: time.Nanosecond,
	}
	appParts, appStructs, stop := deployTestApp(
		istructs.AppQName_test1_app1, 1, false,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			wsb.AddCDoc(featureDoc)
			wsb.AddCommand(vsqlLocalCommand)
			wsb.AddCommand(vsqlImportedCommand)
			wsb.AddCommand(vsqlCancelCommand)
			wsb.AddCommand(istructs.QNameCommandCUD)
			wsb.AddProjector(allProjectorName).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.Types(appdef.TypeKind_Command))
			wsb.AddProjector(specificProjectorName).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.QNames(vsqlLocalCommand))
		},
		func(cfg *istructsmem.AppConfigType) {
			for _, name := range []appdef.QName{vsqlLocalCommand, vsqlImportedCommand, vsqlCancelCommand, istructs.QNameCommandCUD} {
				cfg.Resources.Add(istructsmem.NewCommandFunction(name, istructsmem.NullCommandExec))
			}
			cfg.AddAsyncProjectors(allProjector, specificProjector)
		},
		actCfg)
	stopped := false
	defer func() {
		if !stopped {
			stop()
		}
	}()

	partition := istructs.PartitionID(1)
	idGen := istructsmem.NewIDGenerator()
	createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, partition, istructs.Offset(1), idGen)
	filler := pLogFiller{app: appStructs, partition: partition, offset: 2, cmdQName: command}
	if command == istructs.QNameCommandCUD {
		filler.fillEvent = func(reb istructs.IRawEventBuilder) {
			reb.CUDBuilder().Create(featureDoc).PutRecordID(appdef.SystemField_ID, 1)
		}
	}
	topOffset := filler.fill(1001, idGen)
	appParts.DeployAppPartitions(istructs.AppQName_test1_app1, []istructs.PartitionID{partition})

	require.Eventually(t, func() bool {
		assertions := require.New(t)
		if getActualizerOffset(assertions, appStructs, partition, allProjectorName) < topOffset {
			return false
		}
		if getActualizerOffset(assertions, appStructs, partition, specificProjectorName) < topOffset {
			return false
		}
		if command != vsqlLocalCommand {
			return true
		}
		countsMu.Lock()
		defer countsMu.Unlock()
		return specificCounts[command] > 0
	}, 5*time.Second, time.Millisecond)
	stop()
	stopped = true
	countsMu.Lock()
	defer countsMu.Unlock()
	return allCounts[command], specificCounts[command]
}

// Tests that istructs.Projector offset is updated (flushed) each time after `OffsetFlushRange` items processed
func Test_AsynchronousActualizer_FlushByRange(t *testing.T) {
	require := require.New(t)

	appName, totalPartitions, partitionNr := istructs.AppQName_test1_app1, istructs.NumAppPartitions(2), istructs.PartitionID(2) // test within partition 2

	broker, bCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                2,
		ChannelsPerSubject:      2,
		Subscriptions:           2,
		SubscriptionsPerSubject: 2,
	}, timeu.NewITime())
	defer bCleanup()

	conf := &BasicAsyncActualizerConfig{
		IntentsLimit:  1,
		BundlesLimit:  1,
		FlushInterval: 2 * time.Second,
		Broker:        broker,
	}

	t0 := time.Now()

	appParts, appStructs, stop := deployTestApp(
		appName, totalPartitions, false,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			ProvideViewDef(wsb, incProjectionView, buildProjectionView)
			ProvideViewDef(wsb, decProjectionView, buildProjectionView)
			wsb.AddCommand(testQName)
			wsb.AddProjector(incrementorName).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.QNames(testQName))
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(testIncrementor)
		},
		conf)

	idGen := istructsmem.NewIDGenerator()
	createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1), idGen)
	createWS(appStructs, istructs.WSID(1002), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(2), idGen)

	f := pLogFiller{
		app:       appStructs,
		partition: partitionNr,
		offset:    istructs.Offset(3),
		cmdQName:  testQName,
	}
	f.fill(1001, idGen)
	f.fill(1002, idGen)
	f.fill(1001, idGen)
	f.fill(1001, idGen)
	f.fill(1001, idGen)
	f.fill(1002, idGen)
	f.fill(1001, idGen)
	f.fill(1001, idGen)
	f.fill(1001, idGen)
	topOffset := f.fill(1001, idGen)

	appParts.DeployAppPartitions(appName, []istructs.PartitionID{partitionNr})

	// Wait for the projectors
	for getActualizerOffset(require, appStructs, partitionNr, incrementorName) < topOffset {
		time.Sleep(time.Millisecond)
	}
	require.True(time.Now().Before(t0.Add(conf.FlushInterval)))

	// stop services
	stop()

	// expected projection values
	require.Equal(int32(8), getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(2), getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1002)))
}

// Tests that istructs.Projector offset is updated (flushed) each time after `OffsetFlushInterval`
func Test_AsynchronousActualizer_FlushByInterval(t *testing.T) {
	require := require.New(t)

	appName, totalPartitions, partitionNr := istructs.AppQName_test1_app1, istructs.NumAppPartitions(1), istructs.PartitionID(1) // test within partition 1

	broker, bCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                2,
		ChannelsPerSubject:      2,
		Subscriptions:           2,
		SubscriptionsPerSubject: 2,
	}, timeu.NewITime())
	defer bCleanup()

	actCfg := &BasicAsyncActualizerConfig{
		FlushInterval: 10 * time.Millisecond,
		Broker:        broker,
	}

	t0 := time.Now()

	appParts, appStructs, stop := deployTestApp(
		appName, totalPartitions, false,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			ProvideViewDef(wsb, incProjectionView, buildProjectionView)
			ProvideViewDef(wsb, decProjectionView, buildProjectionView)
			wsb.AddCommand(testQName)
			wsb.AddProjector(incrementorName).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.QNames(testQName))
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(testIncrementor)
		},
		actCfg)

	idGen := istructsmem.NewIDGenerator()
	createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1), idGen)
	createWS(appStructs, istructs.WSID(1002), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(2), idGen)

	f := pLogFiller{
		app:       appStructs,
		partition: partitionNr,
		offset:    istructs.Offset(3),
		cmdQName:  testQName,
	}
	f.fill(1001, idGen)
	f.fill(1002, idGen)
	topOffset := f.fill(1001, idGen)

	appParts.DeployAppPartitions(appName, []istructs.PartitionID{partitionNr})

	// Wait for the projectors
	for getActualizerOffset(require, appStructs, partitionNr, incrementorName) < topOffset {
		time.Sleep(time.Millisecond)
	}
	require.True(time.Now().After(t0.Add(actCfg.FlushInterval)))

	// stop services
	stop()

	// expected projection values
	require.Equal(int32(2), getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(1), getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1002)))
}

func getProjectorsInError(t *testing.T, metrics imetrics.IMetrics, appName appdef.AppQName, vvmName string) *float64 {
	var foundMetricValue float64
	var projInErrors *float64 = nil
	err := metrics.List(func(metric imetrics.IMetric, metricValue float64) (err error) {
		if metric.App() == appName && metric.Vvm() == vvmName && metric.Name() == ProjectorsInError {
			foundMetricValue = metricValue
			projInErrors = &foundMetricValue
		}
		return nil
	})
	require.NoError(t, err)
	return projInErrors
}

// Observes the real reader's transition without changing notification delivery.
type loggingPhaseBroker struct {
	in10n.IN10nBroker
	watching chan struct{}
	once     sync.Once
}

func (b *loggingPhaseBroker) WatchChannel(ctx context.Context, id in10n.ChannelID, notify func(in10n.ProjectionKey, istructs.Offset)) {
	b.once.Do(func() {
		close(b.watching)
	})
	b.IN10nBroker.WatchChannel(ctx, id, notify)
}

type actualizerLoggingTest struct {
	parts         appparts.IAppPartitions
	app           istructs.IAppStructs
	broker        *loggingPhaseBroker
	filler        pLogFiller
	ids           istructs.IIDGenerator
	projector     appdef.QName
	calls         atomic.Int32
	invocationCtx chan context.Context
	stop          func()
}

func newActualizerLoggingTest(t *testing.T, invoke func(istructs.IPLogEvent) error) *actualizerLoggingTest {
	t.Helper()
	return newActualizerLoggingTestWithBuffering(t, invoke, false)
}

func newActualizerLoggingTestWithBuffering(t *testing.T, invoke func(istructs.IPLogEvent) error, buffered bool) *actualizerLoggingTest {
	t.Helper()
	broker, cleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels: 10, ChannelsPerSubject: 10, Subscriptions: 10, SubscriptionsPerSubject: 10,
	}, timeu.NewITime())
	t.Cleanup(cleanup)
	f := &actualizerLoggingTest{
		broker:        &loggingPhaseBroker{IN10nBroker: broker, watching: make(chan struct{})},
		ids:           istructsmem.NewIDGenerator(),
		projector:     appdef.NewQName("test", "LoggingProjector"),
		invocationCtx: make(chan context.Context, 1),
	}
	cmd, args := appdef.NewQName("test", "LoggingCommand"), appdef.NewQName("test", "LoggingArgs")
	cfg := &BasicAsyncActualizerConfig{Broker: f.broker, FlushInterval: time.Millisecond}
	if buffered {
		cfg.FlushInterval = 50 * time.Millisecond
	}
	var stop func()
	f.parts, f.app, stop = deployTestApp(istructs.AppQName_test1_app1, 1, false, testWorkspace, testWorkspaceDescriptor,
		func(ws appdef.IWorkspaceBuilder) {
			ws.AddObject(args).AddField("value", appdef.DataKind_string, false)
			ws.AddCommand(cmd).SetParam(args)
			ws.AddCDoc(appdef.NewQName("test", "LoggingRecord")).AddField("value", appdef.DataKind_float64, false)
			prj := ws.AddProjector(f.projector)
			prj.Events().Add([]appdef.OperationKind{appdef.OperationKind_Execute}, filter.QNames(cmd))
			if !buffered {
				prj.States().Add(sys.Storage_HTTP) // Flush each invocation so saved offsets prove completion.
			}
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(cmd, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(istructs.Projector{Name: f.projector,
				Func: func(event istructs.IPLogEvent, state istructs.IState, _ istructs.IIntents) error {
					select {
					case f.invocationCtx <- state.RequestContext():
					default:
					}
					f.calls.Add(1)
					if invoke != nil {
						return invoke(event)
					}
					return nil
				},
			})
		}, cfg)
	f.stop = sync.OnceFunc(stop)
	t.Cleanup(f.stop)
	f.filler = pLogFiller{app: f.app, partition: 1, offset: 1, cmdQName: cmd,
		fillEvent: func(event istructs.IRawEventBuilder) {
			event.ArgumentObjectBuilder().PutString("value", "private-argument")
			event.CUDBuilder().Create(appdef.NewQName("test", "LoggingRecord")).PutRecordID(appdef.SystemField_ID, 1)
		},
	}
	return f
}

func (f *actualizerLoggingTest) createWorkspace() {
	createWS(f.app, 1001, testWorkspace, testWorkspaceDescriptor, 1, 1, f.ids)
	f.filler.offset = 2
}

func (f *actualizerLoggingTest) start() {
	f.parts.DeployAppPartitions(istructs.AppQName_test1_app1, []istructs.PartitionID{1})
}

func (f *actualizerLoggingTest) waitOffset(require *require.Assertions, offset istructs.Offset) {
	require.Eventually(func() bool {
		actual, err := ActualizerOffset(f.app, 1, f.projector)
		return err == nil && actual >= offset
	},
		5*time.Second, time.Millisecond)
}

func (f *actualizerLoggingTest) waitWatching(require *require.Assertions) {
	require.Eventually(func() bool {
		select {
		case <-f.broker.watching:
			return true
		default:
			return false
		}
	}, 5*time.Second, time.Millisecond)
}

func (f *actualizerLoggingTest) notify(offset istructs.Offset) {
	f.broker.Update(in10n.ProjectionKey{App: istructs.AppQName_test1_app1, Projection: PLogUpdatesQName, WS: 1}, offset)
}

func TestAsyncProjectorEntireLogRunEnd(t *testing.T) {
	t.Run("empty or already flushed log completes immediately", func(t *testing.T) {
		require := require.New(t)
		p := &asyncProjector{}
		done := make(chan struct{})

		_, err := p.DoAsync(t.Context(), &entireLogRunEnd{done: done})
		require.NoError(err)
		select {
		case <-done:
		default:
			require.FailNow("entire log run completion was not signaled")
		}
		require.Nil(p.entireLogRunDone)
	})
	t.Run("unflushed events defer completion", func(t *testing.T) {
		require := require.New(t)
		p := &asyncProjector{pLogOffset: 3}
		done := make(chan struct{})

		_, err := p.DoAsync(t.Context(), &entireLogRunEnd{done: done})
		require.NoError(err)
		select {
		case <-done:
			require.FailNow("entire log run completed before flushing")
		default:
		}
		require.Equal(done, p.entireLogRunDone)
		require.Equal(istructs.Offset(3), p.pLogOffset)
	})
}

func TestAsyncLogging(t *testing.T) {
	t.Run("logging: scn: A new projector logs entire log run start and completion", func(t *testing.T) {
		require := require.New(t)
		// Given projector test.LoggingProjector belongs to test1/app1 partition 1
		// Given a new projector has events through offset 3 and verbose logging is enabled
		logs := logger.StartCapture(t, logger.LogLevelVerbose)
		f := newActualizerLoggingTest(t, nil)
		f.createWorkspace()
		f.filler.fill(1001, f.ids)
		f.filler.fill(1001, f.ids)
		// When the projector finishes the entire log run from FirstOffset
		f.start()
		f.waitWatching(require)
		f.waitOffset(require, 3)
		require.Equal(int32(2), f.calls.Load())
		// Then one INFO ap.entire_log_run.start entry is present
		// And one INFO ap.entire_log_run.complete entry has nonnegative duration_ms
		// And both entries identify the application, projector and partition without message text or args
		// And neither entry includes poffset
		requireEntireLogRunLogs(t, require, logs)
		// And entire log run event details are absent
		logs.NotContains("stage=ap ", "stage=ap.log_cud", "stage=ap.success", "args=", "private-argument")

		// Live notifications resume detailed logging without repeating lifecycle entries.
		offset := f.filler.fill(1001, f.ids)
		f.notify(offset)
		f.waitOffset(require, offset)
		logs.EventuallyHasLine("stage=ap.success", "partid=1", fmt.Sprintf("poffset=%d", offset))
		logs.HasLine("stage=ap ", "partid=1", "triggeredby=test.LoggingCommand", fmt.Sprintf("poffset=%d", offset))
		logs.NotContains("args=", "private-argument")
		requireEntireLogRunLogs(t, require, logs)
	})
	t.Run("logging: scn: Entire log run completion waits for pending processing", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			require := require.New(t)
			logs := logger.StartCapture(t, logger.LogLevelVerbose)
			release := make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			// Given projector test.LoggingProjector belongs to test1/app1 partition 1
			f := newActualizerLoggingTestWithBuffering(t, func(event istructs.IPLogEvent) error {
				if event.PLogOffset() == 3 {
					<-release
				}
				return nil
			}, true)
			t.Cleanup(unblock) // Unblock before stopping the actualizer on assertion failure.
			// Given a new projector has read through offset 3 but its last event is still processing
			f.createWorkspace()
			f.filler.fill(1001, f.ids)
			f.filler.fill(1001, f.ids)
			f.start()
			require.Eventually(func() bool { return f.calls.Load() == 2 }, 5*time.Second, time.Millisecond)
			// Let the reader reach EOF and settle while the final invocation remains blocked.
			synctest.Wait()
			// When the operator inspects the logs
			actual := logs.String()
			// Then ap.entire_log_run.start is present and ap.entire_log_run.complete is absent
			require.Contains(actual, "stage=ap.entire_log_run.start")
			require.NotContains(actual, "stage=ap.entire_log_run.complete")

			// Releasing the last event must allow completion only after its buffered changes are flushed.
			unblock()
			logs.EventuallyHasLine("stage=ap.entire_log_run.complete")
			actualOffset, err := ActualizerOffset(f.app, 1, f.projector)
			require.NoError(err)
			require.Equal(istructs.Offset(3), actualOffset)
			requireEntireLogRunLogs(t, require, logs)
			require.Equal(int32(2), f.calls.Load())
			logs.NotContains("stage=ap ", "stage=ap.log_cud", "stage=ap.success")
		})
	})
	for _, test := range []struct {
		name  string
		level logger.TLogLevel
	}{
		{name: "verbose", level: logger.LogLevelVerbose},
		{name: "info", level: logger.LogLevelInfo},
	} {
		t.Run("logging: scn: An entire log run retry keeps one start and one completion: "+test.name, func(t *testing.T) {
			require := require.New(t)
			logs := logger.StartCapture(t, test.level)
			// Given projector test.LoggingProjector belongs to test1/app1 partition 1
			// Given a new projector fails its first attempt to process offset 2
			var attempts atomic.Int32
			f := newActualizerLoggingTest(t, func(event istructs.IPLogEvent) error {
				if event.PLogOffset() == 2 && attempts.Add(1) == 1 {
					return errors.New("projector failed")
				}
				return nil
			})
			f.createWorkspace()
			f.filler.fill(1001, f.ids)
			// When the entire log run succeeds after an automatic retry
			f.start()
			f.waitWatching(require)
			f.waitOffset(require, 2)
			require.Equal(int32(2), attempts.Load())
			// Then one INFO ap.entire_log_run.start precedes the error and one INFO ap.entire_log_run.complete follows it
			requireEntireLogRunLogs(t, require, logs)
			actual := logs.String()
			require.Less(strings.Index(actual, "stage=ap.entire_log_run.start"), strings.Index(actual, "stage=ap.error"))
			require.Less(strings.Index(actual, "stage=ap.error"), strings.Index(actual, "stage=ap.entire_log_run.complete"))
			// And the ERROR ap.error entry identifies the application, projector, workspace, event and offsets without args
			logs.HasLine("level=ERROR", "stage=ap.error", "projector failed", "vapp=test1/app1", "extension=ap.test.LoggingProjector",
				"partid=1", "wsid=1001", "woffset=0", "poffset=2", "evqname=test.LoggingCommand")
			logs.NotContains("args=", "private-argument")
			logs.NotContains("stage=ap ", "stage=ap.log_cud", "stage=ap.success")
		})
	}
}

func TestAsyncLoggingEntireLogRunTransitions(t *testing.T) {
	t.Run("empty initial log then live events", func(t *testing.T) {
		require := require.New(t)
		logs := logger.StartCapture(t, logger.LogLevelVerbose)
		f := newActualizerLoggingTest(t, nil)
		f.start()
		f.waitWatching(require)
		requireEntireLogRunLogs(t, require, logs)
		logs.NotContains("stage=ap ", "stage=ap.log_cud", "stage=ap.success", "args=", "private-argument")

		f.createWorkspace()
		offset := f.filler.fill(1001, f.ids)
		f.notify(offset)
		f.waitOffset(require, offset)
		logs.EventuallyHasLine("stage=ap.success", "partid=1", fmt.Sprintf("poffset=%d", offset))
		logs.HasLine("stage=ap ", "partid=1", "triggeredby=test.LoggingCommand", fmt.Sprintf("poffset=%d", offset))
		logs.NotContains("args=", "private-argument")
		requireEntireLogRunLogs(t, require, logs)
	})
	t.Run("stopping unfinished entire log run does not log completion", func(t *testing.T) {
		require := require.New(t)
		logs := logger.StartCapture(t, logger.LogLevelInfo)
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		f := newActualizerLoggingTest(t, func(istructs.IPLogEvent) error { <-release; return nil })
		t.Cleanup(unblock)
		f.createWorkspace()
		f.filler.fill(1001, f.ids)
		f.start()
		require.Eventually(func() bool { return f.calls.Load() == 1 }, 5*time.Second, time.Millisecond)
		invocationCtx := <-f.invocationCtx
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			f.stop()
		}()
		require.Eventually(func() bool { return invocationCtx.Err() != nil }, 5*time.Second, time.Millisecond)
		unblock()
		require.Eventually(func() bool {
			select {
			case <-stopped:
				return true
			default:
				return false
			}
		}, 5*time.Second, time.Millisecond)
		logs.HasLine("stage=ap.entire_log_run.start")
		logs.NotContains("stage=ap.entire_log_run.complete")
	})
	t.Run("entire log run completes when all events are skipped", func(t *testing.T) {
		require := require.New(t)
		logs := logger.StartCapture(t, logger.LogLevelInfo)
		f := newActualizerLoggingTest(t, nil)
		f.createWorkspace() // Workspace creation does not trigger this projector.
		f.start()
		f.waitWatching(require)
		requireEntireLogRunLogs(t, require, logs)
		require.Zero(f.calls.Load())
		actual, err := ActualizerOffset(f.app, 1, f.projector)
		require.NoError(err)
		require.Equal(istructs.NullOffset, actual) // The marker must not force a checkpoint.
	})
	t.Run("entire log run stays quiet across read batches", func(t *testing.T) {
		require := require.New(t)
		logs := logger.StartCapture(t, logger.LogLevelVerbose)
		f := newActualizerLoggingTest(t, nil)
		f.createWorkspace()
		for range plogReadBatchSize {
			f.filler.fill(1001, f.ids)
		}
		f.start()
		logs.EventuallyHasLine("stage=ap.entire_log_run.complete")
		expectedOffset := istructs.FirstOffset + istructs.Offset(plogReadBatchSize)
		f.waitOffset(require, expectedOffset)
		require.Equal(int32(plogReadBatchSize), f.calls.Load())
		requireEntireLogRunLogs(t, require, logs)
		logs.NotContains("stage=ap ", "stage=ap.log_cud", "stage=ap.success", "args=")
	})
}

func requireEntireLogRunLogs(t *testing.T, require *require.Assertions, logs logger.ILogCaptor) {
	t.Helper()
	logs.HasLine("level=INFO", "stage=ap.entire_log_run.start")
	logs.HasLine("level=INFO", "stage=ap.entire_log_run.complete")
	require.Equal(1, strings.Count(logs.String(), "stage=ap.entire_log_run.start"))
	require.Equal(1, strings.Count(logs.String(), "stage=ap.entire_log_run.complete"))
	for line := range strings.SplitSeq(logs.String(), "\n") {
		if !strings.Contains(line, "stage=ap.entire_log_run.") {
			continue
		}
		require.Contains(line, "vapp=test1/app1")
		require.Contains(line, "extension=ap.test.LoggingProjector")
		require.Contains(line, "partid=1")
		require.Contains(line, `msg=""`)
		require.NotContains(line, "args=")
		require.NotContains(line, "wsid=")
		require.NotContains(line, "woffset=")
		require.NotContains(line, "poffset=")
		require.NotContains(line, "evqname=")
		if strings.Contains(line, "stage=ap.entire_log_run.complete") {
			require.Regexp(`\bduration_ms=\d+(\s|$)`, line)
		} else {
			require.NotContains(line, "duration_ms=")
		}
	}
}

func TestAsyncLoggingErrorsAndRetry(t *testing.T) {
	for _, test := range []struct {
		name  string
		saved istructs.Offset
		live  bool
	}{
		{name: "saved offset", saved: 1},
		{name: "live events", live: true},
	} {
		for _, level := range []logger.TLogLevel{logger.LogLevelVerbose, logger.LogLevelInfo} {
			t.Run(fmt.Sprintf("%s level %d", test.name, level), func(t *testing.T) {
				require := require.New(t)
				logs := logger.StartCapture(t, level)
				var attempts atomic.Int32
				f := newActualizerLoggingTest(t, func(istructs.IPLogEvent) error {
					if attempts.Add(1) == 1 {
						return errors.New("projector failed")
					}
					return nil
				})
				f.createWorkspace()
				if test.live {
					f.start()
					f.waitWatching(require)
					logs.EventuallyHasLine("stage=ap.entire_log_run.complete")
				} else {
					require.NoError(storeProjectorOffset(f.app, 1, f.projector, test.saved))
				}
				offset := f.filler.fill(1001, f.ids)
				if test.live {
					f.notify(offset)
				} else {
					f.start()
				}
				f.waitOffset(require, offset)
				require.Equal(int32(2), attempts.Load())
				expectedStarts := 1
				if test.saved != istructs.NullOffset {
					expectedStarts = 0
					logs.NotContains("stage=ap.entire_log_run.complete")
				} else {
					logs.EventuallyHasLine("stage=ap.entire_log_run.complete")
					requireEntireLogRunLogs(t, require, logs)
				}
				require.Equal(expectedStarts, strings.Count(logs.String(), "stage=ap.entire_log_run.start"))
				logs.HasLine("level=ERROR", "stage=ap.error", "projector failed", "vapp=test1/app1", "extension=ap.test.LoggingProjector",
					"partid=1", "wsid=1001", "woffset=0", "poffset=2", "evqname=test.LoggingCommand")
				logs.NotContains("args=", "private-argument")
				if level == logger.LogLevelInfo {
					logs.NotContains("stage=ap ", "stage=ap.success")
				} else if !test.live {
					logs.EventuallyHasLine("stage=ap.success", "poffset=2")
				} else {
					logs.HasLine("stage=ap ", "poffset=2", "triggeredby=test.LoggingCommand")
				}
			})
		}
	}
}

func TestAsyncDetailLoggingFailureRetainsEventContext(t *testing.T) {
	require := require.New(t)
	logs := logger.StartCapture(t, logger.LogLevelVerbose)
	f := newActualizerLoggingTest(t, nil)
	f.createWorkspace()
	require.NoError(storeProjectorOffset(f.app, 1, f.projector, 1))
	f.filler.fillEvent = func(event istructs.IRawEventBuilder) {
		event.ArgumentObjectBuilder().PutString("value", "private-argument")
		row := event.CUDBuilder().Create(appdef.NewQName("test", "LoggingRecord"))
		row.PutRecordID(appdef.SystemField_ID, 1)
		row.PutFloat64("value", math.NaN()) // Valid binary value that JSON logging cannot serialize.
	}
	f.filler.fill(1001, f.ids)
	f.start()
	logs.EventuallyHasLine("stage=ap.error", "unsupported value: NaN", "vapp=test1/app1",
		"extension=ap.test.LoggingProjector", "wsid=1001", "woffset=0", "poffset=2", "evqname=test.LoggingCommand")
	require.Zero(f.calls.Load())
	logs.NotContains("args=", "private-argument")
}

func TestAsyncLoggingRetryAfterPartialEntireLogRun(t *testing.T) {
	require := require.New(t)
	logs := logger.StartCapture(t, logger.LogLevelVerbose)
	var attempts atomic.Int32
	f := newActualizerLoggingTest(t, func(event istructs.IPLogEvent) error {
		if event.PLogOffset() == 3 && attempts.Add(1) == 1 {
			return errors.New("projector failed after saving offset 2")
		}
		return nil
	})
	f.createWorkspace()
	f.filler.fill(1001, f.ids)
	f.filler.fill(1001, f.ids)
	f.start()
	logs.EventuallyHasLine("stage=ap.entire_log_run.complete")
	f.waitOffset(require, 3)
	require.Equal(int32(3), f.calls.Load()) // Offset 2 is saved and is not invoked again on retry.
	require.Equal(int32(2), attempts.Load())
	requireEntireLogRunLogs(t, require, logs)
	logs.HasLine("stage=ap.error", "projector failed after saving offset 2", "poffset=3")
	logs.HasLine("stage=ap ", "triggeredby=test.LoggingCommand", "poffset=3")
	logs.HasLine("stage=ap.success", "poffset=3")
	logs.NotContains("args=", "private-argument")
	for line := range strings.SplitSeq(logs.String(), "\n") {
		if strings.Contains(line, "stage=ap ") || strings.Contains(line, "stage=ap.log_cud") || strings.Contains(line, "stage=ap.success") {
			require.Contains(line, "poffset=3") // Initial processing stays quiet; saved-offset retry uses normal logging.
		}
	}
}

func TestAsyncBorrowFailureRetainsEventContext(t *testing.T) {
	for _, level := range []logger.TLogLevel{logger.LogLevelVerbose, logger.LogLevelInfo} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			require := require.New(t)
			logs := logger.StartCapture(t, level)
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			ws := adb.AddWorkspace(testWorkspace)
			ws.AddCommand(testQName)
			ws.AddProjector(incrementorName).Events().Add([]appdef.OperationKind{appdef.OperationKind_Execute}, filter.QNames(testQName))
			prj := appdef.Projector(adb.MustBuild().Type, incrementorName)
			expected := errors.New("partition unavailable")
			ctx := logger.WithContextAttrs(t.Context(), map[string]any{"vapp": "test1/app1", "extension": "ap.test.Projector"})
			parts := &ctxFailingAppParts{failOnCtx: ctx, failErr: expected}
			p := &asyncProjector{iProjector: prj, appParts: parts}
			event := &coreutils.MockPLogEvent{}
			event.On("QName").Return(testQName)
			event.On("Workspace").Return(istructs.WSID(1001))
			event.On("WLogOffset").Return(istructs.Offset(7))
			event.On("Release").Return().Once()
			w := &workpiece{event: event, pLogOffset: 19, logCtx: ctx, entireLogRun: true}
			_, err := p.DoAsync(ctx, w)
			require.ErrorIs(err, expected)
			act := &asyncActualizer{}
			act.Prepare(ctx)
			pipelineErr := pipeline.WireAsyncOperator("Projector", p).NewError(err, w, "test")
			retry, err := act.retrierCfg.OnError(1, time.Millisecond, pipelineErr)
			require.NoError(err)
			require.True(retry)
			logs.HasLine("stage=ap.error", "partition unavailable", "vapp=test1/app1", "extension=ap.test.Projector",
				"wsid=1001", "woffset=7", "poffset=19", "evqname="+testQName.String())
			logs.NotContains("args=")
			event.AssertNotCalled(t, "ArgumentObject")
			event.AssertExpectations(t)
		})
	}
}

func Test_AsynchronousActualizer_Logs(t *testing.T) {
	t.Run("execute projector logs all cuds and success without arguments", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)

		appName, totalPartitions, partitionNr := istructs.AppQName_test1_app1, istructs.NumAppPartitions(1), istructs.PartitionID(1)
		cmdQName := appdef.NewQName("test", "logging_execute_cmd")
		argQName := appdef.NewQName("test", "logging_execute_arg")
		recQName1 := appdef.NewQName("test", "logging_execute_rec1")
		recQName2 := appdef.NewQName("test", "logging_execute_rec2")
		projectorQName := appdef.NewQName("test", "logging_execute_projector")

		broker, cleanup := in10nmem.NewN10nBroker(in10n.Quotas{
			Channels:                2,
			ChannelsPerSubject:      2,
			Subscriptions:           2,
			SubscriptionsPerSubject: 2,
		}, timeu.NewITime())
		defer cleanup()

		actCfg := &BasicAsyncActualizerConfig{
			Broker: broker,
		}

		appParts, appStructs, stop := deployTestApp(
			appName, totalPartitions, false,
			testWorkspace, testWorkspaceDescriptor,
			func(wsb appdef.IWorkspaceBuilder) {
				cmd := wsb.AddCommand(cmdQName)
				cmd.SetParam(argQName)
				wsb.AddObject(argQName).AddField("name", appdef.DataKind_string, false)
				wsb.AddCDoc(recQName1).AddField("name", appdef.DataKind_string, false)
				wsb.AddCDoc(recQName2).AddField("name", appdef.DataKind_string, false)
				wsb.AddProjector(projectorQName).Events().Add(
					[]appdef.OperationKind{appdef.OperationKind_Execute},
					filter.QNames(cmdQName))
			},
			func(cfg *istructsmem.AppConfigType) {
				cfg.Resources.Add(istructsmem.NewCommandFunction(cmdQName, istructsmem.NullCommandExec))
				cfg.AddAsyncProjectors(istructs.Projector{
					Name: projectorQName,
					Func: func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) error { return nil },
				})
			},
			actCfg)
		defer stop()

		idGen := istructsmem.NewIDGenerator()
		createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, partitionNr, istructs.Offset(1), idGen)

		f := pLogFiller{
			app:       appStructs,
			partition: partitionNr,
			offset:    istructs.Offset(2),
			cmdQName:  cmdQName,
			fillEvent: func(reb istructs.IRawEventBuilder) {
				reb.ArgumentObjectBuilder().PutString("name", "hello")
				cuds := reb.CUDBuilder()
				cuds.Create(recQName1).PutRecordID(appdef.SystemField_ID, 1)
				cuds.Create(recQName2).PutRecordID(appdef.SystemField_ID, 2)
			},
		}
		topOffset := f.fill(1001, idGen)

		require.NoError(storeProjectorOffset(appStructs, partitionNr, projectorQName, 1))
		appParts.DeployAppPartitions(appName, []istructs.PartitionID{partitionNr})
		require.Eventually(func() bool {
			actual, err := ActualizerOffset(appStructs, partitionNr, projectorQName)
			return err == nil && actual >= topOffset
		}, 5*time.Second, time.Millisecond)

		logCap.HasLine("stage=ap",
			fmt.Sprintf("vapp=%s", appName),
			fmt.Sprintf("extension=ap.%s", projectorQName),
			"wsid=1001", "woffset=",
			fmt.Sprintf("poffset=%d", topOffset),
			fmt.Sprintf("evqname=%s", cmdQName),

			fmt.Sprintf("triggeredby=%s", cmdQName),
		)
		logCap.HasLine("stage=ap.log_cud", "partid=1", fmt.Sprintf("rectype=%s", recQName1), "op=create",
			`\"sys.IsActive\":true`, fmt.Sprintf(`\"sys.QName\":\"%s\"`, recQName1))
		logCap.HasLine("stage=ap.log_cud", "partid=1", fmt.Sprintf("rectype=%s", recQName2), "op=create",
			`\"sys.IsActive\":true`, fmt.Sprintf(`\"sys.QName\":\"%s\"`, recQName2))
		require.Equal(2, strings.Count(logCap.String(), "op=create"))
		require.Equal(2, strings.Count(logCap.String(), "newfields="))
		logCap.NotContains("oldfields=")
		logCap.EventuallyHasLine("stage=ap.success")
		logCap.NotContains("args=")
	})

	t.Run("record projector logs only triggering cuds", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)

		appName, totalPartitions, partitionNr := istructs.AppQName_test1_app1, istructs.NumAppPartitions(1), istructs.PartitionID(1)
		cmdQName := appdef.NewQName("test", "logging_cud_cmd")
		recQNameLogged := appdef.NewQName("test", "logging_cud_logged")
		recQNameSkipped := appdef.NewQName("test", "logging_cud_skipped")
		projectorQName := appdef.NewQName("test", "logging_cud_projector")

		broker, cleanup := in10nmem.NewN10nBroker(in10n.Quotas{
			Channels:                2,
			ChannelsPerSubject:      2,
			Subscriptions:           2,
			SubscriptionsPerSubject: 2,
		}, timeu.NewITime())
		defer cleanup()

		actCfg := &BasicAsyncActualizerConfig{
			Broker: broker,
		}

		appParts, appStructs, stop := deployTestApp(
			appName, totalPartitions, false,
			testWorkspace, testWorkspaceDescriptor,
			func(wsb appdef.IWorkspaceBuilder) {
				wsb.AddCommand(cmdQName)
				wsb.AddCDoc(recQNameLogged).AddField("name", appdef.DataKind_string, false)
				wsb.AddCDoc(recQNameSkipped).AddField("name", appdef.DataKind_string, false)
				wsb.AddProjector(projectorQName).Events().Add(
					[]appdef.OperationKind{appdef.OperationKind_Insert},
					filter.QNames(recQNameLogged))
			},
			func(cfg *istructsmem.AppConfigType) {
				cfg.Resources.Add(istructsmem.NewCommandFunction(cmdQName, istructsmem.NullCommandExec))
				cfg.AddAsyncProjectors(istructs.Projector{
					Name: projectorQName,
					Func: func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) error { return nil },
				})
			},
			actCfg)
		defer stop()

		idGen := istructsmem.NewIDGenerator()
		createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, partitionNr, istructs.Offset(1), idGen)

		f := pLogFiller{
			app:       appStructs,
			partition: partitionNr,
			offset:    istructs.Offset(2),
			cmdQName:  cmdQName,
			fillEvent: func(reb istructs.IRawEventBuilder) {
				cuds := reb.CUDBuilder()
				cuds.Create(recQNameLogged).PutRecordID(appdef.SystemField_ID, 1)
				cuds.Create(recQNameSkipped).PutRecordID(appdef.SystemField_ID, 2)
			},
		}
		topOffset := f.fill(1001, idGen)

		require.NoError(storeProjectorOffset(appStructs, partitionNr, projectorQName, 1))
		appParts.DeployAppPartitions(appName, []istructs.PartitionID{partitionNr})
		require.Eventually(func() bool {
			actual, err := ActualizerOffset(appStructs, partitionNr, projectorQName)
			return err == nil && actual >= topOffset
		}, 5*time.Second, time.Millisecond)

		logCap.HasLine("stage=ap", fmt.Sprintf("triggeredby=%s", recQNameLogged))
		logCap.HasLine("stage=ap.log_cud", fmt.Sprintf("rectype=%s", recQNameLogged), "op=create",
			`\"sys.IsActive\":true`, fmt.Sprintf(`\"sys.QName\":\"%s\"`, recQNameLogged))
		logCap.NotContains(fmt.Sprintf("rectype=%s", recQNameSkipped))
		require.Equal(1, strings.Count(logCap.String(), "op=create"))
		require.Equal(1, strings.Count(logCap.String(), "newfields="))
		logCap.NotContains("oldfields=")
		logCap.EventuallyHasLine("stage=ap.success")
		logCap.NotContains("args=")
	})
}

// Tests that error is handled correctly.
// Async actualizer should write the error to log, then rebuild and restart itself after retry backoff
func Test_AsynchronousActualizer_ErrorAndRestore(t *testing.T) {
	require := require.New(t)
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)

	appName, totalPartitions, partitionNr := istructs.AppQName_test1_app1, istructs.NumAppPartitions(1), istructs.PartitionID(1) // test within partition 1
	name := appdef.NewQName("test", "failing_projector")

	var attempts atomic.Int32
	restoreCh := make(chan struct{})

	errorsCh := make(chan string, 10)

	broker, cleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                2,
		ChannelsPerSubject:      2,
		Subscriptions:           2,
		SubscriptionsPerSubject: 2,
	}, timeu.NewITime())
	defer cleanup()

	actConf := &BasicAsyncActualizerConfig{
		Broker:        broker,
		BundlesLimit:  10,
		FlushInterval: 10 * time.Millisecond,
	}

	appParts, appStructs, stop := deployTestApp(
		appName, totalPartitions, false,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			ProvideViewDef(wsb, incProjectionView, buildProjectionView)
			ProvideViewDef(wsb, decProjectionView, buildProjectionView)
			wsb.AddCommand(testQName)
			// add not-View and not-Record state to make the projector NonBuffered
			prj := wsb.AddProjector(name)
			prj.Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.QNames(testQName))
			prj.States().Add(sys.Storage_HTTP)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(
				istructs.Projector{
					Name: name,
					Func: func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
						if event.Workspace() == 1002 {
							switch attempts.Add(1) {
							case 1:
								return errors.New("test error")
							case 2:
								<-restoreCh
							}
						}
						return nil
					},
				})
		},
		actConf)

	idGen := istructsmem.NewIDGenerator()
	createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1), idGen)
	createWS(appStructs, istructs.WSID(1002), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(2), idGen)

	f := pLogFiller{
		app:       appStructs,
		partition: partitionNr,
		offset:    istructs.Offset(3),
		cmdQName:  testQName,
	}
	f.fill(1001, idGen)
	f.fill(1002, idGen)
	topOffset := f.fill(1001, idGen)

	appParts.DeployAppPartitions(appName, []istructs.PartitionID{partitionNr})

	logCap.EventuallyHasLine("stage=ap.error",
		fmt.Sprintf("vapp=%s", appName),
		fmt.Sprintf("extension=ap.%s", name),
		"wsid=1002", `msg="test error"`,
	)

	// wait until the istructs.Projector version is updated with the 1st record
	for getActualizerOffset(require, appStructs, partitionNr, name) < istructs.Offset(1) {
		time.Sleep(time.Microsecond)
	}
	require.Eventually(func() bool {
		projInErr := getProjectorsInError(t, actConf.Metrics, appName, actConf.VvmName)
		return projInErr != nil && *projInErr == 1.0
	}, time.Second, 10*time.Millisecond)
	close(restoreCh)

	// Now the istructs.Projector must handle the log till the end
	for getActualizerOffset(require, appStructs, partitionNr, name) < topOffset {
		time.Sleep(time.Microsecond)
	}
	require.Eventually(func() bool {
		projInErr := getProjectorsInError(t, actConf.Metrics, appName, actConf.VvmName)
		return projInErr != nil && *projInErr == 0.0
	}, time.Second, 10*time.Millisecond)

	// stop services
	stop()

	require.EqualValues(2, attempts.Load())

	select {
	case err := <-errorsCh:
		t.Fatal("unexpected error is logged:", err)
	default:
	}
}

func Test_AsynchronousActualizer_ResumeReadAfterNotifications(t *testing.T) {
	require := require.New(t)

	appName, totalPartitions, partitionNr := istructs.AppQName_test1_app1, istructs.NumAppPartitions(1), istructs.PartitionID(1) // test within partition 1

	broker, bCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                2,
		ChannelsPerSubject:      2,
		Subscriptions:           2,
		SubscriptionsPerSubject: 2,
	}, timeu.NewITime())
	defer bCleanup()

	actCfg := &BasicAsyncActualizerConfig{
		IntentsLimit:  2,
		BundlesLimit:  2,
		FlushInterval: 1 * time.Second,
		Broker:        broker,
	}

	appParts, appStructs, stop := deployTestApp(
		appName, totalPartitions, false,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			ProvideViewDef(wsb, incProjectionView, buildProjectionView)
			ProvideViewDef(wsb, decProjectionView, buildProjectionView)
			wsb.AddCommand(testQName)
			wsb.AddProjector(incrementorName).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.QNames(testQName))
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(testIncrementor)
		},
		actCfg)

	idGen := istructsmem.NewIDGenerator()
	createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1), idGen)
	createWS(appStructs, istructs.WSID(1002), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(2), idGen)

	f := pLogFiller{
		app:       appStructs,
		partition: partitionNr,
		offset:    istructs.Offset(3),
		cmdQName:  testQName,
	}
	//Initial events in pLog
	f.fill(1001, idGen)
	topOffset := f.fill(1002, idGen)

	appParts.DeployAppPartitions(appName, []istructs.PartitionID{partitionNr})

	// Wait for the projectors
	for getActualizerOffset(require, appStructs, partitionNr, incrementorName) < topOffset {
		time.Sleep(time.Millisecond)
	}

	//New events in pLog
	f.fill(1001, idGen)
	topOffset = f.fill(1001, idGen)

	//Notify the projectors
	broker.Update(in10n.ProjectionKey{
		App:        appName,
		Projection: PLogUpdatesQName,
		WS:         istructs.WSID(partitionNr),
	}, topOffset)

	// Wait for the projectors
	for getActualizerOffset(require, appStructs, partitionNr, incrementorName) < topOffset {
		time.Sleep(time.Millisecond)
	}

	// stop services
	stop()

	// expected projection values
	require.Equal(int32(3), getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(1), getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1002)))
	projInErrs := getProjectorsInError(t, actCfg.Metrics, appName, actCfg.VvmName)
	require.NotNil(projInErrs)
	require.Equal(0.0, *projInErrs)
}

type pLogFiller struct {
	app       istructs.IAppStructs
	partition istructs.PartitionID
	offset    istructs.Offset
	cmdQName  appdef.QName
	fillEvent func(istructs.IRawEventBuilder)
}

func (f *pLogFiller) fill(wsid istructs.WSID, idGen istructs.IIDGenerator) (offset istructs.Offset) {
	reb := f.app.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		Workspace:         wsid,
		HandlingPartition: f.partition,
		PLogOffset:        f.offset,
		QName:             f.cmdQName,
	})
	if f.fillEvent != nil {
		f.fillEvent(reb)
	}
	rawEvent, err := reb.BuildRawEvent()
	if err != nil {
		panic(err)
	}
	offset = f.offset
	f.offset++
	_, err = f.app.Events().PutPlog(rawEvent, nil, idGen)
	if err != nil {
		panic(err)
	}
	return offset
}

func Test_AsynchronousActualizer_Stress(t *testing.T) {

	/*
		=== RUN   Test_AsynchronousActualizer_Stress
		    async_test.go:594: Total events  : 50000
		    async_test.go:595: Total spent   : 1.108912858s
		    async_test.go:596: Events/sec    : 45089.2057
		    async_test.go:597: One event avg : 22.178µs
		    async_test.go:598: Total batches : 11
		--- PASS: Test_AsynchronousActualizer_Stress (1.70s)
	*/

	/*
		Nikolay Nikitin, 2024-06-27, Windows, amd64, Intel(R) Core(TM) i5-3570 CPU @ 3.40GHz
		=== RUN   Test_AsynchronousActualizer_Stress
		    async_test.go:623: Total events  : 50000
		    async_test.go:624: Total spent   : 2.2667677s
		    async_test.go:625: Events/sec    : 22057.8403
		    async_test.go:626: One event avg : 45.335µs
		    async_test.go:627: Total batches : 20
		--- PASS: Test_AsynchronousActualizer_Stress (2.27s)
	*/

	t.Skip()

	require := require.New(t)

	appName, totalPartitions, partitionNr := istructs.AppQName_test1_app1, istructs.NumAppPartitions(1), istructs.PartitionID(1) // test within partition 1

	broker, bCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                2,
		ChannelsPerSubject:      2,
		Subscriptions:           2,
		SubscriptionsPerSubject: 2,
	}, timeu.NewITime())
	defer bCleanup()

	actMetrics := newSimpleMetrics()

	conf := &BasicAsyncActualizerConfig{
		Broker:    broker,
		AAMetrics: actMetrics,
	}

	t0 := time.Now()

	appParts, appStructs, stop := deployTestApp(
		appName, totalPartitions, false,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			ProvideViewDef(wsb, incProjectionView, buildProjectionView)
			ProvideViewDef(wsb, decProjectionView, buildProjectionView)
			wsb.AddCommand(testQName)
			wsb.AddProjector(incrementorName).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.QNames(testQName))
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(testIncrementor)
		},
		conf)

	idGen := istructsmem.NewIDGenerator()
	createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1), idGen)
	createWS(appStructs, istructs.WSID(1002), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(2), idGen)

	f := pLogFiller{
		app:       appStructs,
		partition: partitionNr,
		offset:    istructs.Offset(3),
		cmdQName:  testQName,
	}

	var topOffset istructs.Offset
	const totalEvents = 50000
	for range totalEvents / 2 {
		f.fill(1001, idGen)
		topOffset = f.fill(1002, idGen)
	}

	appParts.DeployAppPartitions(appName, []istructs.PartitionID{partitionNr})

	// Wait for the projectors
	for actMetrics.value(aaStoredOffset, partitionNr, incrementorName) < int64(topOffset) {
		time.Sleep(time.Millisecond)
	}
	d := time.Since(t0)
	d0 := d.Nanoseconds() / totalEvents
	t.Logf("Total events  : %d", totalEvents)
	t.Logf("Total spent   : %s", d)
	t.Logf("Events/sec    : %.4f", totalEvents/d.Seconds())
	t.Logf("One event avg : %s", time.Duration(d0))
	t.Logf("Total batches : %d", actMetrics.total(aaFlushesTotal))

	// stop services
	stop()

	// expected projection values
	require.Equal(int32(totalEvents/2), getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(totalEvents/2), getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1002)))
}

func Test_AsynchronousActualizer_NonBuffered(t *testing.T) {
	require := require.New(t)

	appName, totalPartitions, partitionNr := istructs.AppQName_test1_app1, istructs.NumAppPartitions(2), istructs.PartitionID(2) // test within partition 2

	broker, bCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                2,
		ChannelsPerSubject:      2,
		Subscriptions:           2,
		SubscriptionsPerSubject: 2,
	}, timeu.NewITime())
	defer bCleanup()

	actMetrics := newSimpleMetrics()

	actCfg := &BasicAsyncActualizerConfig{
		IntentsLimit:  10,
		BundlesLimit:  10,
		FlushInterval: 2 * time.Second,
		Broker:        broker,
		AAMetrics:     actMetrics,
	}

	t0 := time.Now()

	appParts, appStructs, stop := deployTestApp(
		appName, totalPartitions, false,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			ProvideViewDef(wsb, incProjectionView, buildProjectionView)
			ProvideViewDef(wsb, decProjectionView, buildProjectionView)
			wsb.AddCommand(testQName)
			// add not-View and not-Record intent to make the projector NonBuffered
			prj := wsb.AddProjector(incrementorName)
			prj.Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.QNames(testQName))
			prj.Intents().Add(sys.Storage_HTTP)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(testIncrementor)
		},
		actCfg)

	idGen := istructsmem.NewIDGenerator()
	createWS(appStructs, istructs.WSID(1001), testWorkspace, testWorkspaceDescriptor, istructs.PartitionID(1), istructs.Offset(1), idGen)
	f := pLogFiller{
		app:       appStructs,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  testQName,
	}
	f.fill(1001, idGen)
	topOffset := f.fill(1001, idGen)

	appParts.DeployAppPartitions(appName, []istructs.PartitionID{partitionNr})

	// Wait for the projectors
	for actMetrics.value(aaStoredOffset, partitionNr, incrementorName) < int64(topOffset) {
		time.Sleep(time.Millisecond)
	}
	require.True(time.Now().Before(t0.Add(actCfg.FlushInterval))) // no flushes by timer happen

	// stop services
	stop()

	require.EqualValues(2, getProjectionValue(require, appStructs, incProjectionView, istructs.WSID(1001)))
	require.EqualValues(2, actMetrics.value(aaFlushesTotal, partitionNr, incrementorName))
	require.EqualValues(topOffset, actMetrics.value(aaCurrentOffset, partitionNr, incrementorName))
	require.Equal(topOffset, getActualizerOffset(require, appStructs, partitionNr, incrementorName))
}

type testPartition struct {
	number    istructs.PartitionID
	topOffset istructs.Offset
	filler    pLogFiller
}

/*

40 partition x 5 projectors

Before:
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:811: Initialized in 1.045626636s
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:821: Started in 796.852µs
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:846: Actualized 400000 events in 3.779736704s
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:860: Stopped in 312.125µs
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:861: RPS: 105827.48
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:864: PutBatch: 2000000
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:865: Batch Per Second: 529137.39
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:869: FlushesTotal: 1999999


After:
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:849: Initialized in 1.010688701s
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:859: Started in 1.129661ms
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:884: Actualized 400000 events in 2.070089932s
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:903: Stopped in 518.795µs
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:904: RPS: 193228.32
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:907: PutBatch: 1
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:908: Batch Per Second: 0.48
    /home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:912: FlushesTotal: 200

*/

func Test_AsynchronousActualizer_Stress_NonBuffered(t *testing.T) {
	t.Skip()

	require := require.New(t)

	projectorFilter := appdef.NewQName("test", "cmd")
	const totalPartitions = 40
	const projectorsPerPartition = 5
	const eventsPerPartition = 20000

	appName := istructs.AppQName_test1_app1
	partID := make([]istructs.PartitionID, totalPartitions)
	for i := range partID {
		partID[i] = istructs.PartitionID(i)
	}

	prjName := func(i int) appdef.QName {
		return appdef.NewQName("test", fmt.Sprintf("prj_%d", i))
	}

	broker, bCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                totalPartitions * projectorsPerPartition,
		ChannelsPerSubject:      totalPartitions * projectorsPerPartition,
		Subscriptions:           totalPartitions * projectorsPerPartition,
		SubscriptionsPerSubject: totalPartitions * projectorsPerPartition,
	}, timeu.NewITime())
	defer bCleanup()

	actMetrics := newSimpleMetrics()

	actCfg := &BasicAsyncActualizerConfig{
		IntentsLimit:  10,
		BundlesLimit:  10,
		FlushInterval: 2 * time.Second,
		Broker:        broker,
		AAMetrics:     actMetrics,
	}

	t0 := time.Now()

	appParts, appStructs, stop := deployTestApp(
		appName, totalPartitions, true,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			wsb.AddCommand(projectorFilter)
			wsb.AddCommand(testQName)
			for i := range projectorsPerPartition {
				prj := prjName(i)
				wsb.AddProjector(prj).Events().Add(
					[]appdef.OperationKind{appdef.OperationKind_Execute},
					filter.QNames(testQName))
			}
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(projectorFilter, istructsmem.NullCommandExec))
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			for i := range projectorsPerPartition {
				cfg.AddAsyncProjectors(istructs.Projector{
					Name: prjName(i),
					Func: func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) error { return nil },
				})
			}
		},
		actCfg)

	partitions := make([]*testPartition, totalPartitions)
	idGen := istructsmem.NewIDGenerator()

	for i := range partitions {
		pn := istructs.PartitionID(i)
		partitions[i] = &testPartition{
			number: pn,
			filler: pLogFiller{
				app:       appStructs,
				partition: pn,
				offset:    istructs.Offset(1),
				cmdQName:  testQName,
			},
		}
		for j := range eventsPerPartition {
			partitions[i].topOffset = partitions[i].filler.fill(istructs.WSID(j), idGen)
		}
	}

	appParts.DeployAppPartitions(appName, partID)

	t.Logf("Initialized in %s", time.Since(t0))

	// Wait for the projectors
	for i := range totalPartitions {
		tp := partitions[i]
		for k := range projectorsPerPartition {
			stored := actMetrics.value(aaStoredOffset, tp.number, prjName(k))
			for stored < int64(tp.topOffset) {
				time.Sleep(time.Millisecond)
				stored = actMetrics.value(aaStoredOffset, tp.number, prjName(k))
			}
		}
	}

	duration := time.Since(t0)
	totalEvents := totalPartitions * eventsPerPartition
	t.Logf("Actualized %d events in %s ", totalEvents, duration)

	// stop services
	t0 = time.Now()
	stop()

	t.Logf("Stopped in %s ", time.Since(t0))
	t.Logf("RPS: %.2f", float64(totalEvents)/duration.Seconds())
	err := actCfg.Metrics.List(func(m imetrics.IMetric, v float64) error {
		if m.Name() == "voedger_istoragecache_putbatch_total" {
			t.Logf("PutBatch: %.0f", v)
			t.Logf("Batch Per Second: %.2f", v/duration.Seconds())
		}
		return nil
	})
	require.NoError(err)
	t.Logf("FlushesTotal: %d", actMetrics.total(aaFlushesTotal))
}

/*
/home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:1004: Initialized in 1.959437968s
/home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:1014: Started in 504.606µs
/home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:1039: Actualized 800000 events in 2.052865863s
/home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:1058: Stopped in 339.307µs
/home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:1059: RPS: 389699.11
/home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:1062: PutBatch: 1
/home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:1063: Batch Per Second: 0.49
/home/michael/Workspaces/voedger/voedger/pkg/projectors/async_test.go:1067: FlushesTotal: 400
*/
func Test_AsynchronousActualizer_Stress_Buffered(t *testing.T) {
	t.Skip()

	projectorCommand := appdef.NewQName("test", "cmd")
	const totalPartitions = 40
	const projectorsPerPartition = 5
	const eventsPerPartition = 20000

	appName := istructs.AppQName_test1_app1
	partID := make([]istructs.PartitionID, totalPartitions)
	for i := range partID {
		partID[i] = istructs.PartitionID(i)
	}

	prjName := func(i int) appdef.QName {
		return appdef.NewQName("test", fmt.Sprintf("prj_%d", i))
	}

	broker, bCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                totalPartitions * projectorsPerPartition,
		ChannelsPerSubject:      totalPartitions * projectorsPerPartition,
		Subscriptions:           totalPartitions * projectorsPerPartition,
		SubscriptionsPerSubject: totalPartitions * projectorsPerPartition,
	}, timeu.NewITime())
	defer bCleanup()

	actMetrics := newSimpleMetrics()

	actCfg := &BasicAsyncActualizerConfig{
		IntentsLimit:          10,
		BundlesLimit:          10,
		FlushInterval:         1000 * time.Millisecond,
		Broker:                broker,
		AAMetrics:             actMetrics,
		FlushPositionInterval: 10 * time.Second,
	}

	t0 := time.Now()

	appParts, appStructs, stop := deployTestApp(
		appName, totalPartitions, true,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			wsb.AddCommand(projectorCommand)
			wsb.AddCommand(testQName)
			for i := range projectorsPerPartition {
				prj := prjName(i)

				wsb.AddProjector(prj).Events().Add(
					[]appdef.OperationKind{appdef.OperationKind_Execute},
					filter.QNames(projectorCommand))
			}
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(projectorCommand, istructsmem.NullCommandExec))
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			for i := range projectorsPerPartition {
				cfg.AddAsyncProjectors(istructs.Projector{
					Name: prjName(i),
					Func: func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) error { return nil },
				})
			}
		},
		actCfg)

	partitions := make([]*testPartition, totalPartitions)

	idGen := istructsmem.NewIDGenerator()
	for i := range partitions {
		pn := istructs.PartitionID(i)
		partitions[i] = &testPartition{
			number: pn,
			filler: pLogFiller{
				app:       appStructs,
				partition: pn,
				offset:    istructs.Offset(1),
				cmdQName:  testQName,
			},
		}
		for j := range eventsPerPartition {
			partitions[i].topOffset = partitions[i].filler.fill(istructs.WSID(j), idGen)
		}
	}

	appParts.DeployAppPartitions(appName, partID)

	t.Logf("Initialized in %s", time.Since(t0))

	// Wait for the projectors
	for i := range totalPartitions {
		tp := partitions[i]
		for k := range projectorsPerPartition {
			stored := actMetrics.value(aaStoredOffset, tp.number, prjName(k))
			for stored < int64(tp.topOffset) {
				time.Sleep(time.Millisecond)
				stored = actMetrics.value(aaStoredOffset, tp.number, prjName(k))
			}
		}
	}

	duration := time.Since(t0)
	totalEvents := totalPartitions * eventsPerPartition
	t.Logf("Actualized %d events in %s ", totalEvents, duration)

	// stop services
	t0 = time.Now()
	stop()

	t.Logf("Stopped in %s ", time.Since(t0))
	t.Logf("RPS: %.2f", float64(totalEvents)/duration.Seconds())
	err := actCfg.Metrics.List(func(metric imetrics.IMetric, metricValue float64) (err error) {
		if metric.Name() == "voedger_istoragecache_putbatch_total" {
			t.Logf("PutBatch: %.0f", metricValue)
			t.Logf("Batch Per Second: %.2f", metricValue/duration.Seconds())
		}
		return nil
	})
	require.NoError(t, err)
	t.Logf("FlushesTotal: %d", actMetrics.total(aaFlushesTotal))
}

// Reproduces https://untill.atlassian.net/browse/AIR-3888.
//
// Drives asyncActualizer.Run through a transient failure sequence using a flaky
// IAppPartitions decorator:
//   - iter 1: AppDef OK, WaitForBorrow (readOffset) OK, NewChannel sets
//     channelCleanup, then WaitForBorrow (readPlogToTheEnd) fails -> finit
//     terminates the channel.
//   - iter 2: AppDef fails -> init returns BEFORE NewChannel reassigns
//     channelCleanup -> finit invokes the stale closure from iter 1.
//
// Without the `a.channelCleanup = nil` reset after finit() in Run(), the second
// finit() panics in N10nBroker.cleanupChannel with "channel terminated".
func Test_AsynchronousActualizer_ChannelCleanupPanicReproduction(t *testing.T) {
	appName, partitionNr := istructs.AppQName_test1_app1, istructs.PartitionID(1)

	broker, bCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels: 10, ChannelsPerSubject: 10, Subscriptions: 10, SubscriptionsPerSubject: 10,
	}, timeu.NewITime())
	defer bCleanup()

	actCfg := &BasicAsyncActualizerConfig{Broker: broker}
	appParts, _, stop := deployTestApp(appName, 1, false, testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			ProvideViewDef(wsb, incProjectionView, buildProjectionView)
			wsb.AddCommand(testQName)
			wsb.AddProjector(incrementorName).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute}, filter.QNames(testQName))
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(testIncrementor)
		},
		actCfg)
	defer stop()
	appParts.DeployAppPartitions(appName, []istructs.PartitionID{partitionNr})

	flaky := &flakyAppParts{IAppPartitions: appParts}
	act := &asyncActualizer{
		projectorQName: incrementorName,
		conf: AsyncActualizerConf{
			BasicAsyncActualizerConfig: *actCfg, AppQName: appName, PartitionID: partitionNr,
		},
		appParts:   flaky,
		retrierCfg: retrier.NewConfig(time.Millisecond, 5*time.Millisecond),
	}
	vvmCtx, vvmCancel := context.WithCancel(context.Background())
	act.Prepare(vvmCtx)

	panicCh := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				panicCh <- fmt.Sprintf("%v\n%s", r, debug.Stack())
			}
		}()
		act.Run(vvmCtx)
	}()
	defer func() { vvmCancel(); <-done }()

	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	deadline := time.After(5 * time.Second)
	recovered := false
	for !recovered {
		select {
		case p := <-panicCh:
			t.Fatalf("asyncActualizer panicked due to stale channelCleanup:\n%s", p)
		case <-deadline:
			t.Fatal("actualizer did not recover from injected failures")
		case <-tick.C:
			recovered = flaky.appDefCalls.Load() >= 3 && flaky.waitForBorrowCalls.Load() >= 3
		}
	}

	vvmCancel()
	<-done
	select {
	case p := <-panicCh:
		t.Fatalf("asyncActualizer panicked during shutdown:\n%s", p)
	default:
	}
}

type flakyAppParts struct {
	appparts.IAppPartitions
	appDefCalls        atomic.Int32
	waitForBorrowCalls atomic.Int32
}

func (f *flakyAppParts) AppDef(name appdef.AppQName) (appdef.IAppDef, error) {
	if f.appDefCalls.Add(1) == 2 {
		return nil, errors.New("flaky AppDef failure")
	}
	return f.IAppPartitions.AppDef(name)
}

func (f *flakyAppParts) WaitForBorrow(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, kind appparts.ProcessorKind) (appparts.IAppPartition, error) {
	if f.waitForBorrowCalls.Add(1) == 2 {
		return nil, errors.New("flaky WaitForBorrow failure")
	}
	return f.IAppPartitions.WaitForBorrow(ctx, app, partID, kind)
}

// Verifies that when WatchChannel's notify callback triggers a readPlogToOffset that fails,
// the error is recorded as the cause of n10nWatchChannelCtx and surfaced verbatim by keepReading.
func Test_AsynchronousActualizer_KeepReadingPropagatesReadPLogErrorAsCause(t *testing.T) {
	require := require.New(t)
	appName, partitionNr := istructs.AppQName_test1_app1, istructs.PartitionID(1)
	logCap := logger.StartCapture(t, logger.LogLevelError)

	broker, bCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels: 10, ChannelsPerSubject: 10, Subscriptions: 10, SubscriptionsPerSubject: 10,
	}, timeu.NewITime())
	defer bCleanup()

	actCfg := &BasicAsyncActualizerConfig{Broker: broker}
	appParts, _, stop := deployTestApp(appName, 1, false, testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			ProvideViewDef(wsb, incProjectionView, buildProjectionView)
			wsb.AddCommand(testQName)
			wsb.AddProjector(incrementorName).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute}, filter.QNames(testQName))
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			cfg.AddAsyncProjectors(testIncrementor)
		},
		actCfg)
	defer stop()
	appParts.DeployAppPartitions(appName, []istructs.PartitionID{partitionNr})

	injectedErr := errors.New("injected readPlogToOffset failure")
	flaky := &ctxFailingAppParts{IAppPartitions: appParts, failErr: injectedErr}

	act := &asyncActualizer{
		projectorQName: incrementorName,
		conf: AsyncActualizerConf{
			BasicAsyncActualizerConfig: *actCfg, AppQName: appName, PartitionID: partitionNr,
		},
		appParts:   flaky,
		retrierCfg: retrier.NewConfig(time.Millisecond, 5*time.Millisecond),
	}
	vvmCtx := t.Context()

	act.Prepare(vvmCtx)
	require.NoError(act.init(vvmCtx))
	defer act.finit()

	flaky.setFailOnCtx(act.n10nWatchChannelCtx)

	stopNotifier := make(chan struct{})
	notifierDone := make(chan struct{})
	go func() {
		defer close(notifierDone)
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopNotifier:
				return
			case <-ticker.C:
				broker.Update(in10n.ProjectionKey{
					App: appName, Projection: PLogUpdatesQName, WS: istructs.WSID(partitionNr),
				}, istructs.Offset(100))
			}
		}
	}()

	err := act.keepReading(vvmCtx)
	close(stopNotifier)
	<-notifierDone

	require.ErrorIs(err, injectedErr)
	require.Same(injectedErr, context.Cause(act.n10nWatchChannelCtx))
	require.Same(err, context.Cause(act.n10nWatchChannelCtx))

	retry, abortErr := act.retrierCfg.OnError(1, time.Millisecond, err)
	require.True(retry)
	require.NoError(abortErr)
	logCap.HasLine("stage=actualizer.error", "extension=sys._Actualizer", injectedErr.Error())
}

type ctxFailingAppParts struct {
	appparts.IAppPartitions
	mu        sync.Mutex
	failOnCtx context.Context
	failErr   error
}

func (f *ctxFailingAppParts) setFailOnCtx(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failOnCtx = ctx
}

func (f *ctxFailingAppParts) WaitForBorrow(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, kind appparts.ProcessorKind) (appparts.IAppPartition, error) {
	f.mu.Lock()
	failOnCtx := f.failOnCtx
	f.mu.Unlock()
	if failOnCtx != nil && ctx == failOnCtx {
		return nil, f.failErr
	}
	return f.IAppPartitions.WaitForBorrow(ctx, app, partID, kind)
}
