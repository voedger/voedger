/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	istructs "github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/pipeline"
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

	cmdQName := appdef.NewQName("test", "test")
	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewSchema(appDef, incProjectionView, buildProjectionSchema)
			ProvideViewSchema(appDef, decProjectionView, buildProjectionSchema)
			ProvideOffsetsSchema(appDef)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(cmdQName, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1) // test within partition 1

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  cmdQName,
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

	broker := in10nmem.Provide(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	})

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

	cmdQName := appdef.NewQName("test", "test")
	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewSchema(appDef, incProjectionView, buildProjectionSchema)
			ProvideViewSchema(appDef, decProjectionView, buildProjectionSchema)
			ProvideOffsetsSchema(appDef)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(cmdQName, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(2) // test within partition 2

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  cmdQName,
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

	broker := in10nmem.Provide(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	})

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

	cmdQName := appdef.NewQName("test", "test")
	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewSchema(appDef, incProjectionView, buildProjectionSchema)
			ProvideViewSchema(appDef, decProjectionView, buildProjectionSchema)
			ProvideOffsetsSchema(appDef)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(cmdQName, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1) // test within partition 1

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  cmdQName,
	}
	f.fill(1001)
	f.fill(1002)
	topOffset := f.fill(1001)

	withCancel, cancelCtx := context.WithCancel(context.Background())

	broker := in10nmem.Provide(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	})

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

// Tests that error is handled correctly.
// Async actualizer should write the error to log, then rebuild and restart itself after a 30-second pause
func Test_AsynchronousActualizer_ErrorAndRestore(t *testing.T) {
	require := require.New(t)

	name := appdef.NewQName("test", "failing_projector")
	cmdQName := appdef.NewQName("test", "test")
	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewSchema(appDef, incProjectionView, buildProjectionSchema)
			ProvideViewSchema(appDef, decProjectionView, buildProjectionSchema)
			ProvideOffsetsSchema(appDef)
			appDef.Add(name, appdef.DefKind_Object)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(cmdQName, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1) // test within partition 1

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  cmdQName,
	}
	f.fill(1001)
	f.fill(1002)
	topOffset := f.fill(1001)

	withCancel, cancelCtx := context.WithCancel(context.Background())
	errors := make(chan string)
	chanAfterError := make(chan time.Time)

	broker := in10nmem.Provide(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	})

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
		Broker: broker,
	}
	attempts := 0

	factory := func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{Name: name, NonBuffered: true, Func: func(event istructs.IPLogEvent, state istructs.IState, intents istructs.IIntents) (err error) {
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
	require.Equal("error: [test.failing_projector [1] [Projector/doAsync, outWork==nil] test error]", errStr)

	// wait until the istructs.Projector version is updated with the 1st record
	for getActualizerOffset(require, app, partitionNr, name) < istructs.Offset(1) {
		time.Sleep(time.Microsecond)
	}
	require.Equal(1, attempts)

	// tick after-error interval ("30 second delay")
	chanAfterError <- time.Now()

	// Now the istructs.Projector must handle the log till the end
	for getActualizerOffset(require, app, partitionNr, name) < topOffset {
		time.Sleep(time.Microsecond)
	}

	// stop services
	cancelCtx()
	actualizer.Close()

	require.Equal(2, attempts)
}

func Test_AsynchronousActualizer_ResumeReadAfterNotifications(t *testing.T) {
	require := require.New(t)

	cmdQName := appdef.NewQName("test", "test")
	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewSchema(appDef, incProjectionView, buildProjectionSchema)
			ProvideViewSchema(appDef, decProjectionView, buildProjectionSchema)
			ProvideOffsetsSchema(appDef)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(cmdQName, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1) // test within partition 1

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  cmdQName,
	}
	//Initial events in pLog
	f.fill(1001)
	f.fill(1002)

	withCancel, cancelCtx := context.WithCancel(context.Background())

	broker := in10nmem.Provide(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	})

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
	}
	actualizerFactory := ProvideAsyncActualizerFactory()
	actualizer, err := actualizerFactory(conf, incrementorFactory)
	require.NoError(err)

	_ = actualizer.DoSync(conf.Ctx, struct{}{}) // Start service

	time.Sleep(100 * time.Millisecond)

	//New events in pLog
	offset := f.fill(1001)
	f.fill(1001)

	//Notify the projectors
	broker.Update(in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: PlogQName,
		WS:         istructs.WSID(partitionNr),
	}, offset)

	//New events in pLog
	f.fill(1001)
	f.fill(1002)

	//Notify the projectors
	broker.Update(in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: PlogQName,
		WS:         istructs.WSID(partitionNr),
	}, offset)

	//New events in pLog
	f.fill(1001)
	f.fill(1001)

	//Notify the projectors
	broker.Update(in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: PlogQName,
		WS:         istructs.WSID(partitionNr),
	}, offset)

	//New events in pLog
	f.fill(1001)
	topOffset := f.fill(1001)

	//Notify the projectors
	broker.Update(in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: PlogQName,
		WS:         istructs.WSID(partitionNr),
	}, offset)

	// Wait for the projectors
	for getActualizerOffset(require, app, partitionNr, incrementorName) < topOffset {
		time.Sleep(time.Nanosecond)
	}

	// stop services
	cancelCtx()
	actualizer.Close()

	// expected projection values
	require.Equal(int32(8), getProjectionValue(require, app, incProjectionView, istructs.WSID(1001)))
	require.Equal(int32(2), getProjectionValue(require, app, incProjectionView, istructs.WSID(1002)))
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
	generator := func(custom istructs.RecordID, schema appdef.IDef) (storage istructs.RecordID, err error) {
		return istructs.NullRecordID, nil
	}
	_, err = f.app.Events().PutPlog(rawEvent, nil, generator)
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

	cmdQName := appdef.NewQName("test", "test")
	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewSchema(appDef, incProjectionView, buildProjectionSchema)
			ProvideViewSchema(appDef, decProjectionView, buildProjectionSchema)
			ProvideOffsetsSchema(appDef)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(cmdQName, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(1) // test within partition 1

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  cmdQName,
	}

	var topOffset istructs.Offset
	const totalEvents = 50000
	for i := 0; i < totalEvents/2; i++ {
		f.fill(1001)
		topOffset = f.fill(1002)
	}

	withCancel, cancelCtx := context.WithCancel(context.Background())

	broker := in10nmem.Provide(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	})

	metrics := simpleMetrics{}

	// init and launch two actualizers
	actualizerFactory := ProvideAsyncActualizerFactory()
	conf := AsyncActualizerConf{
		Ctx:        withCancel,
		Partition:  partitionNr,
		AppStructs: func() istructs.IAppStructs { return app },
		Broker:     broker,
		Metrics:    &metrics,
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

	cmdQName := appdef.NewQName("test", "test")
	app := appStructs(
		func(appDef appdef.IAppDefBuilder) {
			ProvideViewSchema(appDef, incProjectionView, buildProjectionSchema)
			ProvideViewSchema(appDef, decProjectionView, buildProjectionSchema)
			ProvideOffsetsSchema(appDef)
		},
		func(cfg *istructsmem.AppConfigType) {
			cfg.Resources.Add(istructsmem.NewCommandFunction(cmdQName, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))
		})
	partitionNr := istructs.PartitionID(2) // test within partition 2

	f := pLogFiller{
		app:       app,
		partition: partitionNr,
		offset:    istructs.Offset(1),
		cmdQName:  cmdQName,
	}
	f.fill(1001)
	topOffset := f.fill(1001)

	withCancel, cancelCtx := context.WithCancel(context.Background())

	broker := in10nmem.Provide(in10n.Quotas{
		Channels:               2,
		ChannelsPerSubject:     2,
		Subsciptions:           2,
		SubsciptionsPerSubject: 2,
	})

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
		Metrics:       &metrics,
	}
	actualizerFactory := ProvideAsyncActualizerFactory()
	projectorFactory := func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{Name: incrementorName, NonBuffered: true, Func: incrementor}
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
