/*
 * Copyright (c) 2020-present unTill Pro, Ltd. and Contributors
 */

package edger

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	logger "github.com/heeus/core-logger"
	"github.com/untillpro/voedger/cmd/edger/internal/ctrls"
	"github.com/untillpro/voedger/cmd/edger/internal/metrics"
	"github.com/untillpro/voedger/cmd/edger/internal/states"
)

var signals = make(chan os.Signal, 1)

func runEdger(pars EdgerParams) {
	signal.Notify(signals, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	nodeStates := states.New()

	superController := ctrls.New(
		map[states.AttributeKind]ctrls.MicroControllerFactory{
			//TODO: replace this mocks with real microcontrollers
			states.DockerStackAttribute: ctrls.MockMicroControllerFactory,
			states.CommandAttribute:     ctrls.MockMicroControllerFactory,
			states.EdgerAttribute:       ctrls.MockMicroControllerFactory,
		},
		ctrls.SuperControllerParams{AchievedStateFile: pars.AchievedStateFilePath},
	)

	metricCollectors := metrics.MetricCollectors()
	metricReporters := metrics.MetricReporters()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		Edger(ctx, nodeStates, superController, metricCollectors, metricReporters, pars)
		wg.Done()
	}()

	sig := <-signals
	logger.Info("signal received:", sig)
	cancel()

	wg.Wait()
}

const (
	logEnter = "entered"
	logLeave = "leaved"
)

// getDesiredStateCycle organizes a cycle of requests for the desired state using the `nodeStates` interface.
// Desired states sends (non-blocking write) to `desiredState` channel.
// The cycle continues until the context is interrupted.
func getDesiredStateCycle(
	ctx context.Context,
	nodeStates states.IEdgeNodeState,
	desiredState *states.LastStateChannel[states.DesiredState],
) {
	logger.Info(logEnter)

	desired := states.MakeDesiredState()

	for ctx.Err() == nil {
		if newDesired, err := nodeStates.GetDesiredState(ctx); err == nil {
			if !desired.Equal(newDesired) {
				desired = newDesired.Clone()
				desiredState.Send(newDesired)
			}
		} else {
			logger.Error("can't get desired state: ", err)
		}
	}

	logger.Info(logLeave)
}

// reportActualStateCycle organizes a cycle of reports the actual state using the `nodeStates` interface.
// Reported states receives from `actualState` chanel.
// The cycle continues until the context is interrupted.
func reportActualStateCycle(
	ctx context.Context,
	nodeStates states.IEdgeNodeState,
	actualState *states.LastStateChannel[states.ActualState],
) {
	logger.Info(logEnter)

	actual := states.MakeActualState()

	for ctx.Err() == nil {
		select {
		case newActual := <-actualState.ReceiveChannel():
			if !newActual.Equal(actual) {
				actual = newActual.Clone()
				err := nodeStates.ReportActualState(ctx, newActual)
				if err != nil {
					logger.Error("can't report actual state: ", err)
				}
			}
		case <-ctx.Done():
			break
		}
	}

	logger.Info(logLeave)
}

// superControllerCycle organizes a cycle of achieves the desired state using the `superCtrls` interface.
// Desired states receives from `desiredState` channel.
// Actual states sends to `actualState` channel.
// The cycle continues until the context `ctx` is interrupted.
//   - attemptInterval is the interval between unsuccessful achievements.
func superControllerCycle(
	ctx context.Context,
	superCtrls ctrls.ISuperController,
	desiredState *states.LastStateChannel[states.DesiredState],
	actualState *states.LastStateChannel[states.ActualState],
	attemptInterval time.Duration,
) {
	logger.Info(logEnter)

	desired := states.MakeDesiredState()
	actual := states.MakeActualState()

mainCycle:
	for ctx.Err() == nil {
		select {
		case desired = <-desiredState.ReceiveChannel():
		case <-ctx.Done():
			break mainCycle
		case <-time.After(attemptInterval):
		}

		newActual, err := superCtrls.AchieveState(ctx, desired)
		if err != nil {
			logger.Error("can't achieve desired state: ", err)
		}

		if !actual.Equal(newActual) {
			actual = newActual.Clone()
			actualState.Send(newActual)
		}
	}

	logger.Info(logLeave)
}

// collectMetricsCycle organizes a cycle of collecting the metrics using the `metricCols` interface.
// Collected metrics are placed into `metricsChan` channel.
// The cycle continues until the context is interrupted.
// The interval between calls `metricCols.CollectMetrics()` is set by the `idleInterval` parameter
func collectMetricsCycle(
	ctx context.Context,
	metricCols metrics.IMetricCollectors,
	metricsChan chan<- metrics.Metrics,
	idleInterval time.Duration,
) {
	logger.Info(logEnter)

	for ctx.Err() == nil {
		metrics, err := metricCols.CollectMetrics(ctx)
		if err == nil {
			if metrics != nil {
				metricsChan <- *metrics
			}
		} else {
			logger.Error("can't collect metrics: ", err)
		}
		if !sleepCtx(ctx, idleInterval) {
			break
		}
	}

	logger.Info(logLeave)
}

// reportMetricsCycle organizes a cycle of reporting the metrics using the `metricCols` interface.
// Metrics to report are taken from `metricsChan` channel.
// The cycle continues until the context is interrupted.
// The interval between calls `metricReps.ReportMetrics()` is set by the `idleInterval` parameter
func reportMetricsCycle(
	ctx context.Context,
	metricReps metrics.IMetricReporters,
	metricsChan <-chan metrics.Metrics,
	idleInterval time.Duration,
) {
	logger.Info(logEnter)

	for ctx.Err() == nil {
		select {
		case metrics := <-metricsChan:
			err := metricReps.ReportMetrics(ctx, &metrics)
			if err != nil {
				logger.Error("can't report metrics: ", err)
			}
		case <-ctx.Done():
			break
		default:
			if !sleepCtx(ctx, idleInterval) {
				break
			}
		}
	}

	logger.Info(logLeave)
}

// Returned value is from range [10ms … 1 hour]
func (params EdgerParams) achieveAttemptInterval() (interval time.Duration) {
	const min, max = 10 * time.Millisecond, time.Hour

	interval = params.AchieveAttemptInterval

	switch {
	case interval == 0:
		interval = DefaultAchieveAttemptInterval
	case interval < min:
		interval = min
	case interval > max:
		interval = max
	}
	return interval
}

func Edger(
	ctx context.Context,
	nodeStates states.IEdgeNodeState,
	superCtrls ctrls.ISuperController,
	metricCols metrics.IMetricCollectors,
	metricReps metrics.IMetricReporters,
	params EdgerParams,
) {

	var wg sync.WaitGroup

	func() {
		desiredState := states.NewLastStateChannel[states.DesiredState]()

		wg.Add(1)
		go func() {
			getDesiredStateCycle(ctx, nodeStates, desiredState)
			wg.Done()
		}()

		actualState := states.NewLastStateChannel[states.ActualState]()

		wg.Add(1)
		go func() {
			reportActualStateCycle(ctx, nodeStates, actualState)
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			superControllerCycle(ctx, superCtrls, desiredState, actualState, params.achieveAttemptInterval())
			wg.Done()
		}()

	}()

	func() {
		const metricsChanCapacity = 10
		metricsChan := make(chan metrics.Metrics, metricsChanCapacity)

		wg.Add(1)
		go func() {
			const idleInterval = time.Duration(10 * time.Second)
			collectMetricsCycle(ctx, metricCols, metricsChan, idleInterval)
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			const idleInterval = time.Duration(5 * time.Second)
			reportMetricsCycle(ctx, metricReps, metricsChan, idleInterval)
			wg.Done()
		}()
	}()

	wg.Wait()
}
