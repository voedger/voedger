/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package states

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	logger "github.com/heeus/core-logger"
	"github.com/stretchr/testify/require"
)

func Test_BasicUsage(t *testing.T) {
	states := New()

	ctx, cancel := context.WithCancel(context.Background())

	desired := make(chan *atomic.Value)
	actual := make(chan *atomic.Value)

	superController := func(ds DesiredState) ActualState {
		as := MakeActualState()
		for id, da := range ds {
			as[id] = ActualAttribute{
				Kind:       da.Kind,
				Offset:     da.Offset,
				TimeMs:     time.Now().UnixMilli(),
				AttemptNum: 1,
				Status:     FinishedStatus,
			}
		}
		return as
	}

	wg := sync.WaitGroup{}

	wg.Add(1) // getDesired cycle
	go func() {
		a := atomic.Value{}
		for ctx.Err() == nil {
			if s, err := states.GetDesiredState(ctx); err == nil {
				a.Store(s)
				select {
				case desired <- &a:
				default:
				}
			}
		}
		wg.Done()
	}()

	wg.Add(1) // achieveState cycle
	go func() {
		a := atomic.Value{}
		for ctx.Err() == nil {
			select {
			case d := <-desired:
				if ds, ok := d.Load().(DesiredState); ok {
					as := superController(ds)
					a.Store(as)
					select {
					case actual <- &a:
					default:
					}
				}
			case <-ctx.Done():
				break
			}
		}
		wg.Done()
	}()

	wg.Add(1) // reportActual cycle
	go func() {
		for ctx.Err() == nil {
			select {
			case a := <-actual:
				if as, ok := a.Load().(ActualState); ok {
					_ = states.ReportActualState(ctx, as)
				}
			case <-ctx.Done():
				break
			}
		}
		wg.Done()
	}()

	time.Sleep(1 * time.Second)

	cancel()
	wg.Wait()
}

func TestDesiredState_Clone(t *testing.T) {
	require := require.New(t)

	state := MakeDesiredState()
	const id = "id"
	state[id] = DesiredAttribute{Kind: DockerStackAttribute, Offset: 1, Value: "1.0.0.0", ScheduleTime: time.Now(), Args: []string{"--deploy", "-v"}}

	t.Run("basic usage", func(t *testing.T) {
		s1 := state.Clone()
		require.True(state.Equal(s1))
		require.True(s1.Equal(state))

		const otherID = "other_id"
		s1[otherID] = DesiredAttribute{Kind: CommandAttribute, Offset: 1, Value: "cd .."}
		require.False(state.Equal(s1))
		require.False(s1.Equal(state))
	})

	t.Run("check attribute isolation between copied states", func(t *testing.T) {
		s1 := state.Clone()

		s2 := state.Clone()

		require.True(s1.Equal(s2))

		a1 := s1[id]
		a2 := s2[id]
		require.Equal(a1, a2)
		require.False(&a1 == &a2)
		require.False(&a1.Args == &a2.Args)

		a1.Offset++
		require.True(s1.Equal(s2))
		require.Equal(s1, s2)

		s1[id] = a1
		require.False(s1.Equal(s2))
		require.NotEqual(s1, s2)

		a2.Offset++
		require.False(s2.Equal(s1))
		require.NotEqual(s2, s1)

		s2[id] = a2
		require.True(s2.Equal(s1))
		require.Equal(s2, s1)
	})
}

func TestDesiredState_Equal(t *testing.T) {
	require := require.New(t)

	t.Run("test true if `dst` is receiver", func(t *testing.T) {
		state := MakeDesiredState()
		require.True(state.Equal(state))
	})

	t.Run("test true if both receiver and `dst` are empty", func(t *testing.T) {
		s := MakeDesiredState()
		require.True(s.Equal(MakeDesiredState()))
	})

	t.Run("test false if `dst` has different field", func(t *testing.T) {
		const testKey = "DockerStack-1"
		state := DesiredState{
			testKey: {
				Kind:         DockerStackAttribute,
				Offset:       1,
				ScheduleTime: time.Now(),
				Value:        "ver-1-0-0-0",
				Args:         []string{"--install", "-s"},
			},
		}

		tests := []struct {
			name string
			dst  DesiredState
			want bool
		}{
			{
				name: "true if all is equal",
				dst: DesiredState{
					testKey: state[testKey],
				},
				want: true,
			},
			{
				name: "false if different lengths",
				dst: DesiredState{
					testKey:       state[testKey],
					testKey + "1": state[testKey],
				},
				want: false,
			},
			{
				name: "false if wrong atrribute keys",
				dst: DesiredState{
					testKey + "1": state[testKey],
				},
				want: false,
			},
			{
				name: "false if wrong atrribute",
				dst: DesiredState{
					testKey: {
						Kind:         EdgerAttribute,
						Offset:       state[testKey].Offset,
						ScheduleTime: state[testKey].ScheduleTime,
						Value:        state[testKey].Value,
						Args:         state[testKey].Args,
					},
				},
				want: false,
			},
			{
				name: "false if offset is different",
				dst: DesiredState{
					testKey: {
						Kind:         state[testKey].Kind,
						Offset:       state[testKey].Offset + 1,
						ScheduleTime: state[testKey].ScheduleTime,
						Value:        state[testKey].Value,
						Args:         state[testKey].Args,
					},
				},
				want: false,
			},
			{
				name: "false if schedule time is different",
				dst: DesiredState{
					testKey: {
						Kind:         state[testKey].Kind,
						Offset:       state[testKey].Offset,
						ScheduleTime: state[testKey].ScheduleTime.Add(time.Microsecond),
						Value:        state[testKey].Value,
						Args:         state[testKey].Args,
					},
				},
				want: false,
			},
			{
				name: "false if value is different",
				dst: DesiredState{
					testKey: {
						Kind:         state[testKey].Kind,
						Offset:       state[testKey].Offset,
						ScheduleTime: state[testKey].ScheduleTime,
						Value:        state[testKey].Value + "1",
						Args:         state[testKey].Args,
					},
				},
				want: false,
			},
			{
				name: "false if args count is different",
				dst: DesiredState{
					testKey: {
						Kind:         state[testKey].Kind,
						Offset:       state[testKey].Offset,
						ScheduleTime: state[testKey].ScheduleTime,
						Value:        state[testKey].Value,
						Args:         []string{"--install", "-s", "-r"},
					},
				},
				want: false,
			},
			{
				name: "false if args is different",
				dst: DesiredState{
					testKey: {
						Kind:         state[testKey].Kind,
						Offset:       state[testKey].Offset,
						ScheduleTime: state[testKey].ScheduleTime,
						Value:        state[testKey].Value,
						Args:         []string{"--install", "-t"},
					},
				},
				want: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := state.Equal(tt.dst); got != tt.want {
					t.Errorf("DesiredState.Equal() = %v, want %v", got, tt.want)
				}
				if got := tt.dst.Equal(state); got != tt.want {
					t.Errorf("DesiredState.Equal() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func TestActualState_Clone(t *testing.T) {
	require := require.New(t)

	state := MakeActualState()
	const id = "id"
	state[id] = ActualAttribute{Kind: DockerStackAttribute, Offset: 1, TimeMs: time.Now().UnixMilli(), AttemptNum: 1, Status: FinishedStatus}

	t.Run("basic usage", func(t *testing.T) {
		s1 := state.Clone()
		require.True(state.Equal(s1))
		require.True(s1.Equal(state))

		const otherID = "other_id"
		s1[otherID] = ActualAttribute{Kind: CommandAttribute, Offset: 1, TimeMs: time.Now().UnixMilli(), AttemptNum: 1, Status: PendingStatus}
		require.False(state.Equal(s1))
		require.False(s1.Equal(state))
	})

	t.Run("check attribute isolation", func(t *testing.T) {
		s1 := state.Clone()
		s2 := state.Clone()

		require.True(s1.Equal(s2))

		a1 := s1[id]
		a2 := s2[id]
		require.True(a1 == a2)
		require.False(&a1 == &a2)

		a1.Offset++
		require.True(s1.Equal(s2))
		require.Equal(s1, s2)

		s1[id] = a1
		require.False(s1.Equal(s2))
		require.NotEqual(s1, s2)

		a2.Offset++
		require.False(s2.Equal(s1))
		require.NotEqual(s2, s1)

		s2[id] = a2
		require.True(s2.Equal(s1))
		require.Equal(s2, s1)
	})
}

func TestActualState_Equal(t *testing.T) {
	require := require.New(t)

	t.Run("test true if `dst` is receiver", func(t *testing.T) {
		state := MakeActualState()
		require.True(state.Equal(state))
	})

	t.Run("test true if both receiver and `dst` are empty", func(t *testing.T) {
		s := MakeActualState()
		require.True(s.Equal(MakeActualState()))
	})

	t.Run("test false if `dst` has different field", func(t *testing.T) {
		const testKey = "Happy-Christmas"
		state := ActualState{
			testKey: {
				Kind:       CommandAttribute,
				Offset:     2023,
				TimeMs:     time.Now().UnixMilli(),
				AttemptNum: 1,
				Status:     FinishedStatus,
				Error:      "",
				Info:       "🎄",
			},
		}

		tests := []struct {
			name string
			dst  ActualState
			want bool
		}{
			{
				name: "true if all is equal",
				dst: ActualState{
					testKey: state[testKey],
				},
				want: true,
			},
			{
				name: "false if different lengths",
				dst: ActualState{
					testKey:       state[testKey],
					testKey + "1": state[testKey],
				},
				want: false,
			},
			{
				name: "false if wrong atrributes key",
				dst: ActualState{
					testKey + "1": state[testKey],
				},
				want: false,
			},
			{
				name: "false if different attribute",
				dst: ActualState{
					testKey: {
						Kind:       EdgerAttribute,
						Offset:     state[testKey].Offset,
						TimeMs:     state[testKey].TimeMs,
						AttemptNum: state[testKey].AttemptNum,
						Status:     state[testKey].Status,
						Error:      state[testKey].Error,
						Info:       state[testKey].Info,
					},
				},
				want: false,
			},
			{
				name: "false if offset is different",
				dst: ActualState{
					testKey: {
						Kind:       state[testKey].Kind,
						Offset:     state[testKey].Offset + 1,
						TimeMs:     state[testKey].TimeMs,
						AttemptNum: state[testKey].AttemptNum,
						Status:     state[testKey].Status,
						Error:      state[testKey].Error,
						Info:       state[testKey].Info,
					},
				},
				want: false,
			},
			{
				name: "false if time is different",
				dst: ActualState{
					testKey: {
						Kind:       state[testKey].Kind,
						Offset:     state[testKey].Offset,
						TimeMs:     state[testKey].TimeMs + 1,
						AttemptNum: state[testKey].AttemptNum,
						Status:     state[testKey].Status,
						Error:      state[testKey].Error,
						Info:       state[testKey].Info,
					},
				},
				want: false,
			},
			{
				name: "false if attempt number is different",
				dst: ActualState{
					testKey: {
						Kind:       state[testKey].Kind,
						Offset:     state[testKey].Offset,
						TimeMs:     state[testKey].TimeMs,
						AttemptNum: state[testKey].AttemptNum + 1,
						Status:     state[testKey].Status,
						Error:      state[testKey].Error,
						Info:       state[testKey].Info,
					},
				},
				want: false,
			},
			{
				name: "false if status is different",
				dst: ActualState{
					testKey: {
						Kind:       state[testKey].Kind,
						Offset:     state[testKey].Offset,
						TimeMs:     state[testKey].TimeMs,
						AttemptNum: state[testKey].AttemptNum,
						Status:     PendingStatus,
						Error:      state[testKey].Error,
						Info:       state[testKey].Info,
					},
				},
				want: false,
			},
			{
				name: "false if Error is different",
				dst: ActualState{
					testKey: {
						Kind:       state[testKey].Kind,
						Offset:     state[testKey].Offset,
						TimeMs:     state[testKey].TimeMs,
						AttemptNum: state[testKey].AttemptNum,
						Status:     state[testKey].Status,
						Error:      "error: " + state[testKey].Error,
						Info:       state[testKey].Info,
					},
				},
				want: false,
			},
			{
				name: "false if info is different",
				dst: ActualState{
					testKey: {
						Kind:       state[testKey].Kind,
						Offset:     state[testKey].Offset,
						TimeMs:     state[testKey].TimeMs,
						AttemptNum: state[testKey].AttemptNum,
						Status:     state[testKey].Status,
						Error:      state[testKey].Error,
						Info:       state[testKey].Info + "☦",
					},
				},
				want: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := state.Equal(tt.dst); got != tt.want {
					t.Errorf("ActualState.Equal() = %v, want %v", got, tt.want)
				}
				if got := tt.dst.Equal(state); got != tt.want {
					t.Errorf("ActualState.Equal() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func TestActualState_Achieves(t *testing.T) {
	require := require.New(t)

	t.Run("must be complette if empty states", func(t *testing.T) {
		as := ActualState{}
		ds := DesiredState{}
		require.True(as.Achieves(ds))
	})

	key := "id"
	t.Run("must be complette if empty desired attribute offset", func(t *testing.T) {
		ds := DesiredState{key: {Kind: CommandAttribute, Offset: 0}}
		as := ActualState{key: {Kind: ds[key].Kind, Offset: ds[key].Offset, Status: FinishedStatus}}

		require.True(as.Achieves(ds))
	})

	ds := DesiredState{key: {Kind: DockerStackAttribute, Offset: 1, Value: "1.0.0.0"}}

	t.Run("must be uncomplette if actual state misses attributes", func(t *testing.T) {
		as := ActualState{"otherID": {Kind: ds[key].Kind, Offset: ds[key].Offset, Status: FinishedStatus}}
		require.False(as.Achieves(ds))
	})

	t.Run("must be uncomplette if different kinds", func(t *testing.T) {
		as := ActualState{key: {Kind: CommandAttribute, Offset: ds[key].Offset, Status: FinishedStatus}}
		require.False(as.Achieves(ds))
	})

	t.Run("must be uncomplette if different offsets", func(t *testing.T) {
		as := ActualState{key: {Kind: ds[key].Kind, Offset: ds[key].Offset + 1, Status: FinishedStatus}}
		require.False(as.Achieves(ds))
	})

	t.Run("must be uncomplette if actual is not finished", func(t *testing.T) {
		as := ActualState{key: {Kind: ds[key].Kind, Offset: ds[key].Offset, Status: InProgressStatus}}
		require.False(as.Achieves(ds))
	})

	t.Run("must be uncomplette if actual is finished with error", func(t *testing.T) {
		as := ActualState{key: {Kind: ds[key].Kind, Offset: ds[key].Offset, Status: FinishedStatus, Error: "test error"}}
		require.False(as.Achieves(ds))
	})

	t.Run("must be complette if offsets are equals and actual is success finished", func(t *testing.T) {
		as := ActualState{key: {Kind: ds[key].Kind, Offset: ds[key].Offset, Status: FinishedStatus}}
		require.True(as.Achieves(ds))
	})
}

func TestActualAttribute_Achieves(t *testing.T) {
	require := require.New(t)

	t.Run("must be achieved if empty desired attribute", func(t *testing.T) {
		aa := ActualAttribute{}
		da := DesiredAttribute{}
		require.True(aa.Achieves(da))
	})

	t.Run("must be achieved if empty desired attribute offset", func(t *testing.T) {
		aa := ActualAttribute{Kind: CommandAttribute, Offset: 1, Status: FinishedStatus}
		da := DesiredAttribute{Kind: CommandAttribute, Offset: 0}

		require.True(aa.Achieves(da))
	})

	da := DesiredAttribute{Kind: DockerStackAttribute, Offset: 1, Value: "1.0.0.0"}

	t.Run("must be not achieved if different kinds", func(t *testing.T) {
		aa := ActualAttribute{Kind: CommandAttribute, Offset: da.Offset, Status: FinishedStatus}
		require.False(aa.Achieves(da))
	})

	t.Run("must be not achieved if different offsets", func(t *testing.T) {
		aa := ActualAttribute{Kind: da.Kind, Offset: da.Offset - 1, Status: FinishedStatus}
		require.False(aa.Achieves(da))
	})

	t.Run("must be not achieved if not finished", func(t *testing.T) {
		aa := ActualAttribute{Kind: da.Kind, Offset: da.Offset, Status: InProgressStatus}
		require.False(aa.Achieves(da))
	})

	t.Run("must be not achieved if finished with error", func(t *testing.T) {
		aa := ActualAttribute{Kind: da.Kind, Offset: da.Offset, Status: FinishedStatus, Error: "test error"}
		require.False(aa.Achieves(da))
	})

	t.Run("must be complette if offsets are equals and success finished", func(t *testing.T) {
		aa := ActualAttribute{Kind: da.Kind, Offset: da.Offset, Status: FinishedStatus}
		require.True(aa.Achieves(da))
	})
}

func TestLastStateChannel(t *testing.T) {
	require := require.New(t)

	testState := MakeDesiredState()
	const id = "id"
	testState[id] = DesiredAttribute{Kind: DockerStackAttribute, Offset: 1, Value: "1.0.0.0", ScheduleTime: time.Now(), Args: []string{"--deploy", "-v"}}

	t.Run("basic usage", func(t *testing.T) {
		ch := NewLastStateChannel[DesiredState]()

		s1 := testState.Clone()
		ch.Send(s1)

		s2 := <-ch.ReceiveChannel()
		require.Equal(s1, s2)
		require.Equal(&s1, &s2)
	})

	t.Run("The second method Send() call must replace the state", func(t *testing.T) {
		ch := NewLastStateChannel[DesiredState]()

		s1 := testState.Clone()
		ch.Send(s1)

		s2 := testState.Clone()
		a := s2[id]
		a.Offset++
		s2[id] = a
		ch.Send(s2)

		s3 := <-ch.ReceiveChannel()
		require.Equal(s2, s3)
		require.Equal(&s2, &s3)

		require.NotEqual(s1, s3)
		require.NotEqual(&s1, &s3)
	})

	t.Run("race test", func(t *testing.T) {
		ch := NewLastStateChannel[DesiredState]()

		wg := sync.WaitGroup{}

		var maxOffset AttrOffset = 10

		pulses := 0
		lastState := MakeDesiredState()
		wg.Add(1)
		go func() {
		For:
			for {
				select {
				case s := <-ch.ReceiveChannel():
					pulses++
					a, ok := s[id]
					require.True(ok)
					if a.Offset == maxOffset {
						lastState = s.Clone()
						break For
					}
				default: // non-blocking reading
					time.Sleep(time.Microsecond)
				}
			}
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			for o := AttrOffset(1); o <= maxOffset; o++ {
				s := testState.Clone()
				a := s[id]
				a.Offset = o
				s[id] = a

				ch.Send(s)
				time.Sleep(time.Microsecond)
			}
			wg.Done()
		}()

		wg.Wait()

		require.NotNil(lastState)
		a, ok := lastState[id]
		require.True(ok)
		require.Equal(maxOffset, a.Offset)

		a.Offset = testState[id].Offset
		lastState[id] = a
		require.Equal(lastState, testState)
		require.True(lastState.Equal(testState))

		logger.Info("pulses:", pulses)
	})
}
