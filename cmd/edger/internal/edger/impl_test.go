/*
 * Copyright (c) 2020-present unTill Pro, Ltd. and Contributors
 */

package edger

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	logger "github.com/heeus/core-logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/cmd/edger/internal/metrics"
	"github.com/voedger/voedger/cmd/edger/internal/states"
)

const TestVersion = "0.0.1-alpha"

type mockStates struct {
	mock.Mock
	get    func() (states.DesiredState, error)
	report func(states.ActualState) error
}

func (s *mockStates) GetDesiredState(context.Context) (states.DesiredState, error) {
	return s.get()
}

func (s *mockStates) ReportActualState(ctx context.Context, state states.ActualState) error {
	return s.report(state)
}

func Test_getDesiredStateCycle(t *testing.T) {
	require := require.New(t)

	plan := []states.DesiredState{
		{"docker1": {Kind: states.DockerStackAttribute, Offset: 1, Value: "ver-1-0-0-0"}},

		{"docker1": {Kind: states.DockerStackAttribute, Offset: 1, Value: "ver-1-0-0-0"},
			"docker2": {Kind: states.DockerStackAttribute, Offset: 1, Value: "ver-1-0-0-0"}},

		{"docker1": {Kind: states.DockerStackAttribute, Offset: 1, Value: "ver-1-0-0-0"},
			"docker2": {Kind: states.DockerStackAttribute, Offset: 1, Value: "ver-1-0-0-0"},
			"edger":   {Kind: states.EdgerAttribute, Offset: 1, Value: "ver-1-0-0-0"}},

		{"docker1": {Kind: states.DockerStackAttribute, Offset: 2, Value: "ver-1-0-1-0"},
			"docker2": {Kind: states.DockerStackAttribute, Offset: 1, Value: "ver-1-0-0-0"},
			"edger":   {Kind: states.EdgerAttribute, Offset: 1, Value: "ver-1-0-0-0"}},

		{"docker1": {Kind: states.DockerStackAttribute, Offset: 2, Value: "ver-1-0-1-0"},
			"docker2": {Kind: states.DockerStackAttribute, Offset: 1, Value: "ver-1-0-0-0"},
			"edger":   {Kind: states.EdgerAttribute, Offset: 1, Value: "ver-1-0-0-0"},
			"cmd":     {Kind: states.CommandAttribute, Offset: 1, Value: "dir", Args: []string{"/d"}}},
	}

	t.Run("basic usage", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		step := 0

		finished := atomic.Value{}
		finished.Store(false)

		nodeStates := mockStates{
			get: func() (states.DesiredState, error) {
				if step < len(plan) {
					step++
					logger.Info(step)
					if step == len(plan) {
						finished.Store(true)
					}
					return plan[step-1], nil
				}
				<-ctx.Done()
				return plan[len(plan)-1], nil
			},
			report: nil,
		}

		desiredState := states.NewLastStateChannel[states.DesiredState]()

		wg := sync.WaitGroup{}

		wg.Add(1)
		go func() {
			getDesiredStateCycle(ctx, &nodeStates, desiredState)
			wg.Done()
		}()

		for !finished.Load().(bool) {
			sleepCtx(ctx, time.Millisecond)
		}

		cancel()

		wg.Wait()

		require.Equal(len(plan), step)

		select {
		case final := <-desiredState.ReceiveChannel():
			require.Equal(plan[step-1], final)
		default:
			require.Fail("expected value in `desiredState`, but not founded")
		}
	})

	t.Run("error return available", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		finished := atomic.Value{}
		finished.Store(false)

		nodeStates := mockStates{
			get: func() (states.DesiredState, error) {
				s := states.MakeDesiredState()
				if !finished.Load().(bool) {
					finished.Store(true)
					return s, fmt.Errorf("test error")
				}
				<-ctx.Done()
				return s, nil
			},
			report: nil,
		}
		desiredState := states.NewLastStateChannel[states.DesiredState]()

		wg := sync.WaitGroup{}

		wg.Add(1)
		go func() {
			getDesiredStateCycle(ctx, &nodeStates, desiredState)
			wg.Done()
		}()

		for !finished.Load().(bool) {
			sleepCtx(ctx, time.Duration(1*time.Millisecond))
		}

		cancel()

		wg.Wait()

		select {
		case <-desiredState.ReceiveChannel():
			require.Fail("unexpected value from `desiredState` received")
		default:
		}

	})
}

func Test_reportActualStateCycle(t *testing.T) {
	require := require.New(t)

	now := time.Now().UnixMilli()
	plan := []states.ActualState{
		{"docker1": {Kind: states.DockerStackAttribute, Offset: 1, TimeMs: now, AttemptNum: 1, Status: states.PendingStatus}},
		{"docker1": {Kind: states.DockerStackAttribute, Offset: 1, TimeMs: now + 1, AttemptNum: 1, Status: states.InProgressStatus}},
		{"docker1": {Kind: states.DockerStackAttribute, Offset: 1, TimeMs: now + 2, AttemptNum: 1, Status: states.FinishedStatus, Error: "test error"}},

		{"docker1": {Kind: states.DockerStackAttribute, Offset: 1, TimeMs: now + 3, AttemptNum: 2, Status: states.PendingStatus}},
		{"docker1": {Kind: states.DockerStackAttribute, Offset: 1, TimeMs: now + 4, AttemptNum: 2, Status: states.InProgressStatus}},
		{"docker1": {Kind: states.DockerStackAttribute, Offset: 1, TimeMs: now + 5, AttemptNum: 2, Status: states.FinishedStatus, Info: "test finished"}},
	}

	t.Run("basic usage", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		numOfReports := 0
		lastReported := states.MakeActualState()
		finished := atomic.Value{}

		nodeStates := &mockStates{
			get: nil,
			report: func(state states.ActualState) (err error) {
				if numOfReports < len(plan) {
					numOfReports++
					lastReported = state
					logger.Info("report actual state:", numOfReports)
					if numOfReports == len(plan) {
						finished.Store(true)
					}
				}
				return nil
			},
		}

		actualStates := states.NewLastStateChannel[states.ActualState]()

		wg := sync.WaitGroup{}
		finished.Store(false)

		wg.Add(1)
		go func() {
			reportActualStateCycle(ctx, nodeStates, actualStates)
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			for _, state := range plan {
				actualStates.Send(state)
				sleepCtx(ctx, 1*time.Millisecond)
			}
			wg.Done()
		}()

		for !finished.Load().(bool) {
			sleepCtx(ctx, 1*time.Millisecond)
		}

		cancel()

		wg.Wait()

		require.GreaterOrEqual(len(plan), numOfReports)
		require.Equal(plan[len(plan)-1], lastReported)
	})

	t.Run("error return available", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		numOfReports := 0
		finished := atomic.Value{}

		nodeStates := &mockStates{
			get: nil,
			report: func(state states.ActualState) (err error) {
				numOfReports++
				if state.Equal(plan[len(plan)-1]) {
					finished.Store(true)
				}
				return fmt.Errorf("test error")
			},
		}

		actualStates := states.NewLastStateChannel[states.ActualState]()

		wg := sync.WaitGroup{}
		finished.Store(false)

		wg.Add(1)
		go func() {
			reportActualStateCycle(ctx, nodeStates, actualStates)
			wg.Done()
		}()

		for _, state := range plan {
			actualStates.Send(state)
			sleepCtx(ctx, 1*time.Millisecond)
		}

		for !finished.Load().(bool) {
			sleepCtx(ctx, 1*time.Millisecond)
		}

		cancel()

		wg.Wait()

		require.LessOrEqual(1, numOfReports)
	})
}

type mockSuperController struct {
	mock.Mock
	achieve func(desired states.DesiredState) (states.ActualState, error)
}

func (c *mockSuperController) AchieveState(ctx context.Context, desired states.DesiredState) (states.ActualState, error) {
	return c.achieve(desired)
}

type mockMetricCollectors struct {
	mock.Mock
	collect func() (*metrics.Metrics, error)
}

func (m *mockMetricCollectors) CollectMetrics(context.Context) (*metrics.Metrics, error) {
	return m.collect()
}

type mockMetricReporters struct {
	mock.Mock
	report func(*metrics.Metrics) error
}

func (m *mockMetricReporters) ReportMetrics(ctx context.Context, metric *metrics.Metrics) error {
	return m.report(metric)
}

func Test_controllerCycle(t *testing.T) {
	require := require.New(t)

	plan := []states.DesiredState{
		{"docker": {Kind: states.DockerStackAttribute, Offset: 1, Value: "ver-1-0-0-0"}},

		{"docker": {Kind: states.DockerStackAttribute, Offset: 1, Value: "ver-1-0-0-0"},
			"edger": {Kind: states.EdgerAttribute, Offset: 1, Value: "ver-1-0-0-0"}},

		{"docker": {Kind: states.DockerStackAttribute, Offset: 1, Value: "ver-1-0-0-0"},
			"edger": {Kind: states.EdgerAttribute, Offset: 1, Value: "ver-1-0-0-0"},
			"cmd":   {Kind: states.CommandAttribute, Offset: 1, Value: "dir"}},

		{"docker": {Kind: states.DockerStackAttribute, Offset: 2, Value: "ver-1-0-1-0"},
			"edger": {Kind: states.EdgerAttribute, Offset: 1, Value: "ver-1-0-0-0"},
			"cmd":   {Kind: states.CommandAttribute, Offset: 1, Value: "dir"}},
	}

	t.Run("basic usage", func(t *testing.T) {
		numOfAchieves := 0
		lastActual := states.MakeActualState()

		finished := atomic.Value{}

		microControllers := &mockSuperController{
			achieve: func(desired states.DesiredState) (actual states.ActualState, err error) {
				numOfAchieves++
				changes := 0
				actual = lastActual.Clone()
				for id, da := range desired {
					if da.Offset == 0 {
						continue
					}

					aa, ok := actual[id]
					if !ok {
						aa = states.ActualAttribute{Kind: da.Kind}
					}
					if aa.Offset != da.Offset {
						aa.Offset = da.Offset
						aa.AttemptNum = 1
						aa.Status = states.InProgressStatus
						aa.TimeMs = time.Now().UnixMilli()
						actual[id] = aa
						changes++
						continue
					}
					if aa.Status != states.FinishedStatus {
						aa.Status = states.FinishedStatus
						aa.TimeMs = time.Now().UnixMilli()
						actual[id] = aa
						changes++
						continue
					}
				}
				if changes > 0 {
					lastActual = actual.Clone()
				}
				if desired.Equal(plan[len(plan)-1]) && actual.Achieves(desired) {
					finished.Store(true)
				}
				return actual, nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())

		desiredState := states.NewLastStateChannel[states.DesiredState]()
		actualState := states.NewLastStateChannel[states.ActualState]()

		numOfAchieves = 0
		finished.Store(false)

		wg := sync.WaitGroup{}

		wg.Add(1)
		go func() {
			superControllerCycle(ctx, microControllers, desiredState, actualState, time.Duration(1*time.Millisecond))
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			for step := 0; step < len(plan); step++ {
				desiredState.Send(plan[step])
				sleepCtx(ctx, 10*time.Millisecond)
			}
			wg.Done()
		}()

		for !finished.Load().(bool) {
			time.Sleep(1 * time.Microsecond)
		}

		cancel()

		wg.Wait()

		logger.Info("numOfAchieves:", numOfAchieves)
		require.Greater(numOfAchieves, 0)
		require.True(lastActual.Achieves(plan[len(plan)-1]))
	})

	t.Run("error return available", func(t *testing.T) {
		finished := atomic.Value{}

		microControllers := &mockSuperController{
			achieve: func(desired states.DesiredState) (actual states.ActualState, err error) {
				finished.Store(true)
				return nil, fmt.Errorf("test error")
			},
		}

		ctx, cancel := context.WithCancel(context.Background())

		desiredState := states.NewLastStateChannel[states.DesiredState]()
		actualState := states.NewLastStateChannel[states.ActualState]()

		finished.Store(false)

		wg := sync.WaitGroup{}

		wg.Add(1)
		go func() {
			superControllerCycle(ctx, microControllers, desiredState, actualState, time.Duration(1*time.Millisecond))
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			for step := 0; step < len(plan); step++ {
				desiredState.Send(plan[step])
				sleepCtx(ctx, 10*time.Millisecond)
			}
			wg.Done()
		}()

		for !finished.Load().(bool) {
			time.Sleep(1 * time.Microsecond)
		}

		cancel()

		wg.Wait()

		select {
		case <-actualState.ReceiveChannel():
			require.Fail("unexpected value from `actualState` received")
		default:
		}
	})

}

func TestEdger_EmptyCycle(t *testing.T) {

	t.Run("empty edger cycle two seconds", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		nodeStates := &mockStates{
			get:    func() (states.DesiredState, error) { return states.MakeDesiredState(), nil },
			report: func(states.ActualState) error { return nil },
		}

		microControllers := &mockSuperController{
			achieve: func(states.DesiredState) (states.ActualState, error) { return states.MakeActualState(), nil },
		}

		metricCollectors := &mockMetricCollectors{
			collect: func() (*metrics.Metrics, error) { return nil, nil },
		}

		metricReporters := &mockMetricReporters{
			report: func(*metrics.Metrics) error { return nil },
		}

		wg := sync.WaitGroup{}

		wg.Add(1)
		go func() {
			Edger(ctx, nodeStates, microControllers, metricCollectors, metricReporters, EdgerParams{})
			wg.Done()
		}()

		time.Sleep(2 * time.Second)
		cancel()

		wg.Wait()
	})
}

func TestEdger_DeployDockerStack(t *testing.T) {
	require := require.New(t)

	t.Run("success deploy docker stack", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		key := "docker"
		plan := []states.DesiredState{
			{key: {Kind: states.DockerStackAttribute, Offset: 1, Value: "ver-1-0-0-0"}},
		}

		numOfDesires := 0
		numOfAchieves := 0
		numOfReports := 0

		finished := atomic.Value{}
		finished.Store(false)

		lastDesired := states.MakeDesiredState()
		lastReported := states.MakeActualState()
		nodeStates := &mockStates{
			get: func() (state states.DesiredState, err error) {
				if int(numOfDesires) < len(plan) {
					state = plan[numOfDesires]
					numOfDesires++
					logger.Info("get desired state:", numOfDesires)
					lastDesired = state
					return state, nil
				}
				<-ctx.Done()
				return plan[len(plan)-1], nil
			},
			report: func(state states.ActualState) (err error) {
				for id, aa := range state {
					da, ok := plan[0][id]
					require.True(ok)
					require.Equal(aa.Kind, da.Kind)
					require.Equal(aa.Offset, da.Offset)
					if aa.Status == states.FinishedStatus {
						finished.Store(true)
					}
				}
				numOfReports++
				logger.Info("report actual state:", numOfReports)
				lastReported = state
				return nil
			},
		}

		lastAchieved := states.MakeActualState()
		microControllers := &mockSuperController{
			achieve: func(desired states.DesiredState) (states.ActualState, error) {
				achieved := lastAchieved.Clone()
				for id, da := range desired {
					if da.Offset == 0 {
						continue
					}

					aa, ok := achieved[id]
					if !ok {
						aa = states.ActualAttribute{Kind: da.Kind}
					}

					if aa.Offset != da.Offset {
						aa.Offset = da.Offset
						aa.AttemptNum = 1
						aa.Status = states.UndefinedStatus
					}

					if aa.Status < states.FinishedStatus {
						aa.Status++
						aa.TimeMs = time.Now().UnixMilli()
					}

					achieved[id] = aa
				}
				lastAchieved = achieved.Clone()
				numOfAchieves++
				logger.Info("achieve state:", numOfAchieves)
				return achieved, nil
			},
		}

		metricCollectors := &mockMetricCollectors{
			collect: func() (*metrics.Metrics, error) { return nil, nil },
		}

		metricReporters := &mockMetricReporters{
			report: func(*metrics.Metrics) error { return nil },
		}

		wg := sync.WaitGroup{}
		finished.Store(false)

		wg.Add(1)
		go func() {
			Edger(ctx, nodeStates, microControllers, metricCollectors, metricReporters, EdgerParams{})
			wg.Done()
		}()

		for !finished.Load().(bool) {
			time.Sleep(1 * time.Millisecond)
		}
		cancel()

		wg.Wait()

		require.EqualValues(len(plan), numOfDesires)
		t.Run("check last desired state is planned", func(t *testing.T) {
			da := lastDesired[key]
			require.Equal(da, plan[len(plan)-1][key])
		})

		require.LessOrEqual(3*len(plan), numOfAchieves)
		require.True(lastAchieved.Achieves(plan[len(plan)-1]), "last achieved state is not desired")

		require.LessOrEqual(len(plan), numOfReports)
		require.Equal(lastAchieved, lastReported, "last reported state is not equal to last archived state")
	})
}
