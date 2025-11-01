/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package actualizers

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	imetrics "github.com/voedger/voedger/pkg/metrics"
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

// Tests that error is handled correctly.
// Async actualizer should write the error to log, then rebuild and restart itself after a 30-second pause
func Test_AsynchronousActualizer_ErrorAndRestore(t *testing.T) {
	require := require.New(t)

	appName, totalPartitions, partitionNr := istructs.AppQName_test1_app1, istructs.NumAppPartitions(1), istructs.PartitionID(1) // test within partition 1
	name := appdef.NewQName("test", "failing_projector")

	attempts := 0

	errorsCh := make(chan string, 10)

	broker, cleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                2,
		ChannelsPerSubject:      2,
		Subscriptions:           2,
		SubscriptionsPerSubject: 2,
	}, timeu.NewITime())
	defer cleanup()

	actConf := &BasicAsyncActualizerConfig{
		Broker: broker,

		LogError: func(args ...interface{}) {
			errorsCh <- fmt.Sprint("error: ", args)
		},

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
							if attempts == 0 {
								attempts++
								return errors.New("test error")
							}
							attempts++
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

	// Wait for the logged error
	errStr := <-errorsCh

	require.Equal("error: [test.failing_projector [1] wsid[1002] offset[0]: test error]", errStr)

	// wait until the istructs.Projector version is updated with the 1st record
	for getActualizerOffset(require, appStructs, partitionNr, name) < istructs.Offset(1) {
		time.Sleep(time.Microsecond)
	}
	require.Equal(1, attempts)
	projInErr := getProjectorsInError(t, actConf.Metrics, appName, actConf.VvmName)
	require.NotNil(projInErr)
	require.Equal(1.0, *projInErr)

	// Now the istructs.Projector must handle the log till the end
	for getActualizerOffset(require, appStructs, partitionNr, name) < topOffset {
		time.Sleep(time.Microsecond)
	}
	projInErr = getProjectorsInError(t, actConf.Metrics, appName, actConf.VvmName)
	require.NotNil(projInErr)
	require.Equal(0.0, *projInErr)

	// stop services
	stop()

	require.Equal(2, attempts)

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
}

func (f *pLogFiller) fill(wsid istructs.WSID, idGen istructs.IIDGenerator) (offset istructs.Offset) {
	reb := f.app.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			Workspace:         wsid,
			HandlingPartition: f.partition,
			PLogOffset:        f.offset,
			QName:             f.cmdQName,
		},
	})
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
	for i := 0; i < totalEvents/2; i++ {
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
		LogError: func(args ...interface{}) {
			require.Fail("actualizer error", args...)
		},
	}

	t0 := time.Now()

	appParts, appStructs, stop := deployTestApp(
		appName, totalPartitions, true,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			wsb.AddCommand(projectorFilter)
			wsb.AddCommand(testQName)
			for i := 0; i < projectorsPerPartition; i++ {
				prj := prjName(i)
				wsb.AddProjector(prj).Events().Add(
					[]appdef.OperationKind{appdef.OperationKind_Execute},
					filter.QNames(testQName))
			}
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(projectorFilter, istructsmem.NullCommandExec))
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			for i := 0; i < projectorsPerPartition; i++ {
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
		for j := 0; j < eventsPerPartition; j++ {
			partitions[i].topOffset = partitions[i].filler.fill(istructs.WSID(j), idGen)
		}
	}

	appParts.DeployAppPartitions(appName, partID)

	t.Logf("Initialized in %s", time.Since(t0))

	// Wait for the projectors
	for i := 0; i < totalPartitions; i++ {
		tp := partitions[i]
		for k := 0; k < projectorsPerPartition; k++ {
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
		LogError:              func(args ...interface{}) {},
		FlushPositionInterval: 10 * time.Second,
	}

	t0 := time.Now()

	appParts, appStructs, stop := deployTestApp(
		appName, totalPartitions, true,
		testWorkspace, testWorkspaceDescriptor,
		func(wsb appdef.IWorkspaceBuilder) {
			wsb.AddCommand(projectorCommand)
			wsb.AddCommand(testQName)
			for i := 0; i < projectorsPerPartition; i++ {
				prj := prjName(i)

				wsb.AddProjector(prj).Events().Add(
					[]appdef.OperationKind{appdef.OperationKind_Execute},
					filter.QNames(projectorCommand))
			}
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(projectorCommand, istructsmem.NullCommandExec))
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
			for i := 0; i < projectorsPerPartition; i++ {
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
		for j := 0; j < eventsPerPartition; j++ {
			partitions[i].topOffset = partitions[i].filler.fill(istructs.WSID(j), idGen)
		}
	}

	appParts.DeployAppPartitions(appName, partID)

	t.Logf("Initialized in %s", time.Since(t0))

	// Wait for the projectors
	for i := 0; i < totalPartitions; i++ {
		tp := partitions[i]
		for k := 0; k < projectorsPerPartition; k++ {
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
