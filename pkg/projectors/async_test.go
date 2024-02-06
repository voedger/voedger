/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/state"
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

	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewDef(appDef, incProjectionView, buildProjectionView)
			ProvideViewDef(appDef, decProjectionView, buildProjectionView)
			appDef.AddCommand(testQName)
			appDef.AddProjector(incrementorName).AddEvent(testQName, appdef.ProjectorEventKind_Execute)
			appDef.AddProjector(decrementorName).AddEvent(testQName, appdef.ProjectorEventKind_Execute)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1) // test within partition 1

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  testQName,
	}
	f.fill(1001)
	f.fill(1002)
	f.fill(1001)
	f.fill(1001)
	f.fill(1001)
	f.fill(1002)
	f.fill(1001)
	f.fill(1001)
	f.fill(1001)
	topOffset := f.fill(1001)

	withCancel, cancelCtx := context.WithCancel(context.Background())

	// store the initial actualizer offsets
	//
	// 1. there will be no stored offset for incrementor, so it starts
	// from the beginning of the log
	//
	// 2. Decrementor will have the offset=4 stored (will start from
	// 5th (index 4 in pLog array)):
	_ = storeProjectorOffset(app, partitionNr, decrementorName, istructs.Offset(4))

	broker, cleanup := in10nmem.ProvideEx2(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	}, time.Now)
	defer cleanup()

	// init and launch two actualizers
	actualizers := make([]pipeline.ISyncOperator, 2)
	actualizerFactory := ProvideAsyncActualizerFactory()
	for i, factory := range []istructs.ProjectorFactory{incrementorFactory, decrementorFactory} {
		conf := AsyncActualizerConf{
			Ctx:        withCancel,
			Partition:  partitionNr,
			AppStructs: func() istructs.IAppStructs { return app },
			Broker:     broker,
		}
		actualizer, err := actualizerFactory(conf, factory)
		require.NoError(err)
		require.NoError(actualizer.DoSync(conf.Ctx, struct{}{})) // Start service
		actualizers[i] = actualizer
	}

	// Wait for the projectors
	for getActualizerOffset(require, app, partitionNr, incrementorName) < topOffset {
		time.Sleep(time.Nanosecond)
	}
	for getActualizerOffset(require, app, partitionNr, decrementorName) < topOffset {
		time.Sleep(time.Nanosecond)
	}
	// stop services
	cancelCtx()
	for i := range actualizers {
		actualizers[i].Close()
	}

	// expected projection values
	require.Equal(int32(8), getProjectionValue(require, app, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(2), getProjectionValue(require, app, incProjectionView, istructs.WSID(1002)))
	require.Equal(int32(-5), getProjectionValue(require, app, decProjectionView, istructs.WSID(1001)))
	require.Equal(int32(-1), getProjectionValue(require, app, decProjectionView, istructs.WSID(1002)))
}

// Tests that istructs.Projector offset is updated (flushed) each time after `OffsetFlushRange` items processed
func Test_AsynchronousActualizer_FlushByRange(t *testing.T) {
	require := require.New(t)

	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewDef(appDef, incProjectionView, buildProjectionView)
			ProvideViewDef(appDef, decProjectionView, buildProjectionView)
			appDef.AddCommand(testQName)
			appDef.AddProjector(incrementorName).AddEvent(testQName, appdef.ProjectorEventKind_Execute)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(2) // test within partition 2

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  testQName,
	}
	f.fill(1001)
	f.fill(1002)
	f.fill(1001)
	f.fill(1001)
	f.fill(1001)
	f.fill(1002)
	f.fill(1001)
	f.fill(1001)
	f.fill(1001)
	topOffset := f.fill(1001)

	withCancel, cancelCtx := context.WithCancel(context.Background())

	broker, cleanup := in10nmem.ProvideEx2(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	}, time.Now)
	defer cleanup()

	// init and launch actualizer
	conf := AsyncActualizerConf{
		Ctx:           withCancel,
		Partition:     partitionNr,
		AppStructs:    func() istructs.IAppStructs { return app },
		IntentsLimit:  1,
		BundlesLimit:  1,
		FlushInterval: 2 * time.Second,
		Broker:        broker,
	}
	actualizerFactory := ProvideAsyncActualizerFactory()
	actualizer, err := actualizerFactory(conf, incrementorFactory)
	require.NoError(err)

	t0 := time.Now()
	err = actualizer.DoSync(conf.Ctx, struct{}{}) // Start service
	require.NoError(err)

	// Wait for the projectors
	for getActualizerOffset(require, app, partitionNr, incrementorName) < topOffset {
		time.Sleep(time.Nanosecond)
	}
	require.True(time.Now().Before(t0.Add(conf.FlushInterval)))
	// stop services
	cancelCtx()
	actualizer.Close()

	// expected projection values
	require.Equal(int32(8), getProjectionValue(require, app, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(2), getProjectionValue(require, app, incProjectionView, istructs.WSID(1002)))
}

// Tests that istructs.Projector offset is updated (flushed) each time after `OffsetFlushInterval`
func Test_AsynchronousActualizer_FlushByInterval(t *testing.T) {
	require := require.New(t)

	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewDef(appDef, incProjectionView, buildProjectionView)
			ProvideViewDef(appDef, decProjectionView, buildProjectionView)
			appDef.AddCommand(testQName)
			appDef.AddProjector(incrementorName).AddEvent(testQName, appdef.ProjectorEventKind_Execute)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1) // test within partition 1

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  testQName,
	}
	f.fill(1001)
	f.fill(1002)
	topOffset := f.fill(1001)

	withCancel, cancelCtx := context.WithCancel(context.Background())

	broker, cleanup := in10nmem.ProvideEx2(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	}, time.Now)
	defer cleanup()

	// init and launch actualizer
	conf := AsyncActualizerConf{
		Ctx:           withCancel,
		Partition:     partitionNr,
		AppStructs:    func() istructs.IAppStructs { return app },
		FlushInterval: 10 * time.Millisecond,
		Broker:        broker,
	}
	actualizerFactory := ProvideAsyncActualizerFactory()
	actualizer, err := actualizerFactory(conf, incrementorFactory)
	require.NoError(err)

	t0 := time.Now()
	err = actualizer.DoSync(conf.Ctx, struct{}{}) // Start service
	require.NoError(err)

	// Wait for the projectors
	for getActualizerOffset(require, app, partitionNr, incrementorName) < topOffset {
		time.Sleep(time.Nanosecond)
	}
	require.True(time.Now().After(t0.Add(conf.FlushInterval)))
	// stop services
	cancelCtx()
	actualizer.Close()

	// expected projection values
	require.Equal(int32(2), getProjectionValue(require, app, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(1), getProjectionValue(require, app, incProjectionView, istructs.WSID(1002)))
}

func getProjectorsInError(metrics imetrics.IMetrics, appName istructs.AppQName, vvmName string) *float64 {
	var foundMetricValue float64
	var projInErrors *float64 = nil
	metrics.List(func(metric imetrics.IMetric, metricValue float64) (err error) {
		if metric.App() == appName && metric.Vvm() == vvmName && metric.Name() == ProjectorsInError {
			foundMetricValue = metricValue
			projInErrors = &foundMetricValue
		}
		return nil
	})
	return projInErrors
}

// Tests that error is handled correctly.
// Async actualizer should write the error to log, then rebuild and restart itself after a 30-second pause
func Test_AsynchronousActualizer_ErrorAndRestore(t *testing.T) {
	require := require.New(t)

	name := appdef.NewQName("test", "failing_projector")
	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewDef(appDef, incProjectionView, buildProjectionView)
			ProvideViewDef(appDef, decProjectionView, buildProjectionView)
			appDef.AddCommand(testQName)
			// add not-View and not-Record state to make the projector NonBuffered
			appDef.AddProjector(name).AddEvent(testQName, appdef.ProjectorEventKind_Execute).AddState(state.Http)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1) // test within partition 1

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  testQName,
	}
	f.fill(1001)
	f.fill(1002)
	topOffset := f.fill(1001)

	withCancel, cancelCtx := context.WithCancel(context.Background())
	errors := make(chan string, 10)
	chanAfterError := make(chan time.Time)

	broker, cleanup := in10nmem.ProvideEx2(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	}, time.Now)
	defer cleanup()

	metrics := imetrics.Provide()

	// init and launch actualizer
	conf := AsyncActualizerConf{
		Ctx:        withCancel,
		Partition:  partitionNr,
		AppStructs: func() istructs.IAppStructs { return app },
		AfterError: func(d time.Duration) <-chan time.Time {
			if d.Seconds() != 30.0 {
				panic("unexpected pause")
			}
			return chanAfterError
		},
		BundlesLimit:  10,
		FlushInterval: 10 * time.Millisecond,
		LogError: func(args ...interface{}) {
			errors <- fmt.Sprint("error: ", args)
		},
		Broker:   broker,
		Metrics:  metrics,
		VvmName:  "test",
		AppQName: istructs.AppQName_test1_app1,
	}
	attempts := 0

	factory := func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{Name: name, Func: func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
			if event.Workspace() == 1002 {
				if attempts == 0 {
					attempts++
					return fmt.Errorf("test error") // First attempt will fail
				}
				attempts++
			}
			return nil
		}}
	}

	actualizerFactory := ProvideAsyncActualizerFactory()
	actualizer, err := actualizerFactory(conf, factory)
	require.NoError(err)
	require.NoError(actualizer.DoSync(conf.Ctx, struct{}{})) // Start service

	// Wait for the logged error
	errStr := <-errors
	require.Equal("error: [test.failing_projector [1] wsid[1002] offset[0]: test error]", errStr)

	// wait until the istructs.Projector version is updated with the 1st record
	for getActualizerOffset(require, app, partitionNr, name) < istructs.Offset(1) {
		time.Sleep(time.Microsecond)
	}
	require.Equal(1, attempts)
	projInErr := getProjectorsInError(metrics, istructs.AppQName_test1_app1, "test")
	require.NotNil(projInErr)
	require.InDelta(1.0, *projInErr, 0.0001)

	// tick after-error interval ("30 second delay")
	chanAfterError <- time.Now()

	// Now the istructs.Projector must handle the log till the end
	for getActualizerOffset(require, app, partitionNr, name) < topOffset {
		time.Sleep(time.Microsecond)
	}
	projInErr = getProjectorsInError(metrics, istructs.AppQName_test1_app1, "test")
	require.NotNil(projInErr)
	require.InDelta(0.0, *projInErr, 0.0001)

	// stop services
	cancelCtx()
	actualizer.Close()

	require.Equal(2, attempts)
}

func Test_AsynchronousActualizer_ResumeReadAfterNotifications(t *testing.T) {
	require := require.New(t)

	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewDef(appDef, incProjectionView, buildProjectionView)
			ProvideViewDef(appDef, decProjectionView, buildProjectionView)
			appDef.AddCommand(testQName)
			appDef.AddProjector(incrementorName).AddEvent(testQName, appdef.ProjectorEventKind_Execute)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1) // test within partition 1

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  testQName,
	}
	//Initial events in pLog
	f.fill(1001)
	topOffset := f.fill(1002)

	withCancel, cancelCtx := context.WithCancel(context.Background())
	metrics := imetrics.Provide()

	broker, cleanup := in10nmem.ProvideEx2(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	}, time.Now)
	defer cleanup()

	// init and launch actualizer
	conf := AsyncActualizerConf{
		Ctx:           withCancel,
		AppQName:      istructs.AppQName_test1_app1,
		Partition:     partitionNr,
		AppStructs:    func() istructs.IAppStructs { return app },
		IntentsLimit:  2,
		BundlesLimit:  2,
		FlushInterval: 1 * time.Second,
		Broker:        broker,
		VvmName:       "test",
		Metrics:       metrics,
	}
	actualizerFactory := ProvideAsyncActualizerFactory()
	actualizer, err := actualizerFactory(conf, incrementorFactory)
	require.NoError(err)

	_ = actualizer.DoSync(conf.Ctx, struct{}{}) // Start service

	// Wait for the projectors
	for getActualizerOffset(require, app, partitionNr, incrementorName) < topOffset {
		time.Sleep(time.Nanosecond)
	}

	//New events in pLog
	f.fill(1001)
	topOffset = f.fill(1001)

	//Notify the projectors
	broker.Update(in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: PLogUpdatesQName,
		WS:         istructs.WSID(partitionNr),
	}, topOffset)

	// Wait for the projectors
	for getActualizerOffset(require, app, partitionNr, incrementorName) < topOffset {
		time.Sleep(time.Nanosecond)
	}

	// stop services
	cancelCtx()
	actualizer.Close()

	// expected projection values
	require.Equal(int32(3), getProjectionValue(require, app, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(1), getProjectionValue(require, app, incProjectionView, istructs.WSID(1002)))
	projInErrs := getProjectorsInError(metrics, istructs.AppQName_test1_app1, "test")
	require.NotNil(projInErrs)
	require.InDelta(0.0, *projInErrs, 0.0001)
}

type pLogFiller struct {
	app       istructs.IAppStructs
	partition istructs.PartitionID
	offset    istructs.Offset
	cmdQName  appdef.QName
}

func (f *pLogFiller) fill(WSID istructs.WSID) (offset istructs.Offset) {
	reb := f.app.Events().GetNewRawEventBuilder(istructs.NewRawEventBuilderParams{
		GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
			Workspace:         WSID,
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
	_, err = f.app.Events().PutPlog(rawEvent, nil, istructsmem.NewIDGenerator())
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
	t.Skip()

	require := require.New(t)

	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewDef(appDef, incProjectionView, buildProjectionView)
			ProvideViewDef(appDef, decProjectionView, buildProjectionView)
			appDef.AddCommand(testQName)
			appDef.AddProjector(incrementorName).AddEvent(testQName, appdef.ProjectorEventKind_Execute)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1) // test within partition 1

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  testQName,
	}

	var topOffset istructs.Offset
	const totalEvents = 50000
	for i := 0; i < totalEvents/2; i++ {
		f.fill(1001)
		topOffset = f.fill(1002)
	}

	withCancel, cancelCtx := context.WithCancel(context.Background())

	broker, cleanup := in10nmem.ProvideEx2(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	}, time.Now)
	defer cleanup()

	metrics := simpleMetrics{}

	// init and launch two actualizers
	actualizerFactory := ProvideAsyncActualizerFactory()
	conf := AsyncActualizerConf{
		Ctx:        withCancel,
		Partition:  partitionNr,
		AppStructs: func() istructs.IAppStructs { return app },
		Broker:     broker,
		AAMetrics:  &metrics,
	}
	actualizer, err := actualizerFactory(conf, incrementorFactory)
	require.NoError(err)
	require.NoError(actualizer.DoSync(conf.Ctx, struct{}{})) // Start service

	t0 := time.Now()
	// Wait for the projectors
	for atomic.LoadInt64(&metrics.storedOffset) < int64(topOffset) {
		time.Sleep(time.Nanosecond)
	}
	d := time.Since(t0)
	d0 := d.Nanoseconds() / totalEvents
	t.Logf("Total events  : %d", totalEvents)
	t.Logf("Total spent   : %s", d)
	t.Logf("Events/sec    : %.4f", totalEvents/d.Seconds())
	t.Logf("One event avg : %s", time.Duration(d0))
	t.Logf("Total batches : %d", metrics.flushesTotal)

	// stop services
	cancelCtx()
	actualizer.Close()

	// expected projection values
	require.Equal(int32(totalEvents/2), getProjectionValue(require, app, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(totalEvents/2), getProjectionValue(require, app, incProjectionView, istructs.WSID(1002)))

}

type simpleMetrics struct {
	flushesTotal  int64
	currentOffset int64
	storedOffset  int64
}

func (m *simpleMetrics) Increase(metricName string, partition istructs.PartitionID, projection appdef.QName, valueDelta float64) {
	if metricName == aaCurrentOffset {
		atomic.AddInt64(&m.currentOffset, int64(valueDelta))
	} else if metricName == aaFlushesTotal {
		atomic.AddInt64(&m.flushesTotal, int64(valueDelta))
	} else {
		atomic.AddInt64(&m.storedOffset, int64(valueDelta))
	}
}

func (m *simpleMetrics) Set(metricName string, partition istructs.PartitionID, projection appdef.QName, value float64) {
	if metricName == aaCurrentOffset {
		atomic.StoreInt64(&m.currentOffset, int64(value))
	} else if metricName == aaFlushesTotal {
		atomic.StoreInt64(&m.flushesTotal, int64(value))
	} else {
		atomic.StoreInt64(&m.storedOffset, int64(value))
	}
}

func Test_AsynchronousActualizer_NonBuffered(t *testing.T) {
	require := require.New(t)

	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewDef(appDef, incProjectionView, buildProjectionView)
			ProvideViewDef(appDef, decProjectionView, buildProjectionView)
			appDef.AddCommand(testQName)
			// add not-View and not-Record intent to make the projector NonBuffered
			appDef.AddProjector(incrementorName).AddEvent(testQName, appdef.ProjectorEventKind_Execute).AddIntent(state.Http)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(2) // test within partition 2

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  testQName,
	}
	f.fill(1001)
	topOffset := f.fill(1001)

	withCancel, cancelCtx := context.WithCancel(context.Background())

	broker, cleanup := in10nmem.ProvideEx2(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	}, time.Now)
	defer cleanup()

	metrics := simpleMetrics{}

	// init and launch actualizer
	conf := AsyncActualizerConf{
		Ctx:           withCancel,
		Partition:     partitionNr,
		AppStructs:    func() istructs.IAppStructs { return app },
		IntentsLimit:  10,
		BundlesLimit:  10,
		FlushInterval: 2 * time.Second,
		Broker:        broker,
		AAMetrics:     &metrics,
	}
	actualizerFactory := ProvideAsyncActualizerFactory()
	projectorFactory := func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{Name: incrementorName, Func: incrementor}
	}
	actualizer, err := actualizerFactory(conf, projectorFactory)
	require.NoError(err)

	t0 := time.Now()
	err = actualizer.DoSync(conf.Ctx, struct{}{}) // Start service
	require.NoError(err)

	// Wait for the projectors
	for atomic.LoadInt64(&metrics.storedOffset) < int64(topOffset) {
		time.Sleep(time.Nanosecond)
	}
	require.True(time.Now().Before(t0.Add(conf.FlushInterval))) // no flushes by timer happen
	// stop services
	cancelCtx()
	actualizer.Close()

	require.Equal(int32(2), getProjectionValue(require, app, incProjectionView, istructs.WSID(1001)))
	require.Equal(int64(2), metrics.flushesTotal)
	require.Equal(int64(topOffset), metrics.currentOffset)
	require.Equal(topOffset, getActualizerOffset(require, app, partitionNr, incrementorName))
}

type testActualizerCtx struct {
	op      pipeline.ISyncOperator
	metrics *simpleMetrics
}

type testPartition struct {
	number      istructs.PartitionID
	topOffset   istructs.Offset
	filler      pLogFiller
	actualizers []testActualizerCtx
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

	projectorFilter := appdef.NewQName("pkg", "fake")
	const totalPartitions = 40
	const projectorsPerPartition = 5
	const eventsPerPartition = 10000

	app := appStructsCached(
		func(appDef appdef.IAppDefBuilder) {
			appDef.AddCommand(projectorFilter)
			appDef.AddCommand(testQName)
			appDef.AddProjector(incrementorName).AddEvent(projectorFilter, appdef.ProjectorEventKind_Execute)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		})
	partitions := make([]*testPartition, totalPartitions)

	withCancel, cancelCtx := context.WithCancel(context.Background())

	broker, cleanup := in10nmem.ProvideEx2(in10n.Quotas{
		Channels:               totalPartitions * projectorsPerPartition,
		ChannelsPerSubject:     totalPartitions * projectorsPerPartition,
		Subsciptions:           totalPartitions * projectorsPerPartition,
		SubsciptionsPerSubject: totalPartitions * projectorsPerPartition,
	}, time.Now)
	defer cleanup()

	actualizerFactory := ProvideAsyncActualizerFactory()
	t0 := time.Now()

	var wg sync.WaitGroup

	for i := range partitions {
		pn := istructs.PartitionID(i)
		partitions[i] = &testPartition{
			number:      pn,
			actualizers: make([]testActualizerCtx, projectorsPerPartition),
			filler: pLogFiller{
				app:       app,
				partition: pn,
				offset:    istructs.Offset(1),
				cmdQName:  testQName,
			},
		}
		for j := 0; j < eventsPerPartition; j++ {
			partitions[i].topOffset = partitions[i].filler.fill(istructs.WSID(j))
		}
		for k := 0; k < projectorsPerPartition; k++ {
			wg.Add(1)
			k := k
			i := i
			go func() {
				defer wg.Done()
				metrics := simpleMetrics{}

				conf := AsyncActualizerConf{
					Ctx:           withCancel,
					Partition:     pn,
					AppStructs:    func() istructs.IAppStructs { return app },
					IntentsLimit:  10,
					BundlesLimit:  10,
					FlushInterval: 2 * time.Second,
					Broker:        broker,
					AAMetrics:     &metrics,
					LogError:      func(args ...interface{}) {},
				}

				projectorFactory := func(partition istructs.PartitionID) istructs.Projector {
					return istructs.Projector{
						Name: incrementorName,
						Func: incrementor,
					}
				}
				actualizer, err := actualizerFactory(conf, projectorFactory)
				require.NoError(err)

				partitions[i].actualizers[k] = testActualizerCtx{
					op:      actualizer,
					metrics: &metrics,
				}

			}()
		}
	}
	wg.Wait()
	t.Logf("Initialized in %s", time.Since(t0))

	// init and launch actualizer
	t0 = time.Now()
	for i := range partitions {
		for k := 0; k < projectorsPerPartition; k++ {
			err := partitions[i].actualizers[k].op.DoSync(withCancel, struct{}{})
			require.NoError(err)
		}
	}
	t.Logf("Started in %s", time.Since(t0))
	t0 = time.Now()

	// Wait for the projectors
	for {
		complete := true
		for i := 0; i < totalPartitions && complete; i++ {
			tp := partitions[i]
			for k := 0; k < projectorsPerPartition && complete; k++ {
				ts := &tp.actualizers[k]
				stored := atomic.LoadInt64(&ts.metrics.storedOffset)
				for stored < int64(tp.topOffset) {
					complete = false
					break
				}
			}
		}
		if complete {
			break
		}
		time.Sleep(time.Nanosecond)
	}

	duration := time.Since(t0)
	totalEvents := totalPartitions * eventsPerPartition
	t.Logf("Actualized %d events in %s ", totalEvents, duration)
	// PutBatch calls
	t0 = time.Now()

	flushesTotal := 0
	for i := range partitions {
		for k := 0; k < projectorsPerPartition; k++ {
			flushesTotal += int(partitions[i].actualizers[k].metrics.flushesTotal)
		}
	}

	// stop services
	cancelCtx()
	for i := range partitions {
		for k := 0; k < projectorsPerPartition; k++ {
			partitions[i].actualizers[k].op.Close()
		}
	}

	t.Logf("Stopped in %s ", time.Since(t0))
	t.Logf("RPS: %.2f", float64(totalEvents)/duration.Seconds())
	metrics.List(func(metric imetrics.IMetric, metricValue float64) (err error) {
		if metric.Name() == "voedger_istoragecache_putbatch_total" {
			t.Logf("PutBatch: %.0f", metricValue)
			t.Logf("Batch Per Second: %.2f", metricValue/duration.Seconds())
		}
		return nil
	})
	t.Logf("FlushesTotal: %d", flushesTotal)
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
	require := require.New(t)

	projectorFilter := appdef.NewQName("pkg", "fake")
	const totalPartitions = 40
	const projectorsPerPartition = 5
	const eventsPerPartition = 20000

	app := appStructsCached(
		func(appDef appdef.IAppDefBuilder) {
			appDef.AddCommand(projectorFilter)
			appDef.AddCommand(testQName)
			appDef.AddProjector(incrementorName).AddEvent(projectorFilter, appdef.ProjectorEventKind_Execute)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(testQName, istructsmem.NullCommandExec))
		})
	partitions := make([]*testPartition, totalPartitions)

	withCancel, cancelCtx := context.WithCancel(context.Background())

	broker, cleanup := in10nmem.ProvideEx2(in10n.Quotas{
		Channels:               totalPartitions * projectorsPerPartition,
		ChannelsPerSubject:     totalPartitions * projectorsPerPartition,
		Subsciptions:           totalPartitions * projectorsPerPartition,
		SubsciptionsPerSubject: totalPartitions * projectorsPerPartition,
	}, time.Now)
	defer cleanup()

	actualizerFactory := ProvideAsyncActualizerFactory()
	t0 := time.Now()

	var wg sync.WaitGroup

	for i := range partitions {
		pn := istructs.PartitionID(i)
		partitions[i] = &testPartition{
			number:      pn,
			actualizers: make([]testActualizerCtx, projectorsPerPartition),
			filler: pLogFiller{
				app:       app,
				partition: pn,
				offset:    istructs.Offset(1),
				cmdQName:  testQName,
			},
		}
		for j := 0; j < eventsPerPartition; j++ {
			partitions[i].topOffset = partitions[i].filler.fill(istructs.WSID(j))
		}
		for k := 0; k < projectorsPerPartition; k++ {
			wg.Add(1)
			k := k
			i := i
			go func() {
				defer wg.Done()
				metrics := simpleMetrics{}

				conf := AsyncActualizerConf{
					Ctx:                   withCancel,
					Partition:             pn,
					AppStructs:            func() istructs.IAppStructs { return app },
					IntentsLimit:          10,
					BundlesLimit:          10,
					FlushInterval:         1000 * time.Millisecond,
					Broker:                broker,
					AAMetrics:             &metrics,
					LogError:              func(args ...interface{}) {},
					FlushPositionInverval: 10 * time.Second,
				}

				projectorFactory := func(partition istructs.PartitionID) istructs.Projector {
					return istructs.Projector{
						Name: incrementorName,
						Func: incrementor,
						// EventsFilter: []appdef.QName{projectorFilter},
					}
				}
				actualizer, err := actualizerFactory(conf, projectorFactory)
				require.NoError(err)

				partitions[i].actualizers[k] = testActualizerCtx{
					op:      actualizer,
					metrics: &metrics,
				}

			}()
		}
	}
	wg.Wait()
	t.Logf("Initialized in %s", time.Since(t0))

	// init and launch actualizer
	t0 = time.Now()
	for i := range partitions {
		for k := 0; k < projectorsPerPartition; k++ {
			err := partitions[i].actualizers[k].op.DoSync(withCancel, struct{}{})
			require.NoError(err)
		}
	}
	t.Logf("Started in %s", time.Since(t0))
	t0 = time.Now()

	// Wait for the projectors
	for {
		complete := true
		for i := 0; i < totalPartitions && complete; i++ {
			tp := partitions[i]
			for k := 0; k < projectorsPerPartition && complete; k++ {
				ts := &tp.actualizers[k]
				stored := atomic.LoadInt64(&ts.metrics.storedOffset)
				for stored < int64(tp.topOffset) {
					complete = false
					break
				}
			}
		}
		if complete {
			break
		}
		time.Sleep(time.Nanosecond)
	}

	duration := time.Since(t0)
	totalEvents := totalPartitions * eventsPerPartition
	t.Logf("Actualized %d events in %s ", totalEvents, duration)
	// PutBatch calls
	t0 = time.Now()

	flushesTotal := 0
	for i := range partitions {
		for k := 0; k < projectorsPerPartition; k++ {
			flushesTotal += int(partitions[i].actualizers[k].metrics.flushesTotal)
		}
	}

	// stop services
	cancelCtx()
	for i := range partitions {
		for k := 0; k < projectorsPerPartition; k++ {
			partitions[i].actualizers[k].op.Close()
		}
	}

	t.Logf("Stopped in %s ", time.Since(t0))
	t.Logf("RPS: %.2f", float64(totalEvents)/duration.Seconds())
	metrics.List(func(metric imetrics.IMetric, metricValue float64) (err error) {
		if metric.Name() == "voedger_istoragecache_putbatch_total" {
			t.Logf("PutBatch: %.0f", metricValue)
			t.Logf("Batch Per Second: %.2f", metricValue/duration.Seconds())
		}
		return nil
	})
	t.Logf("FlushesTotal: %d", flushesTotal)
}
