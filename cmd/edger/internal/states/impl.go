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
)

// Clone method creates new state, copies all keys and state attributes to result
func (s *DesiredState) Clone() (dst DesiredState) {
	dst = MakeDesiredState()
	for id, a := range *s {
		dst[id] = a
	}
	return dst
}

// Equal returns is desired state equals to specified state `dst`
func (s *DesiredState) Equal(dst DesiredState) bool {

	equalArgs := func(a1, a2 *[]string) bool {
		if len(*a1) != len(*a2) {
			return false
		}
		for i := 0; i < len(*a1); i++ {
			if (*a1)[i] != (*a2)[i] {
				return false
			}
		}
		return true
	}

	if len(*s) != len(dst) {
		return false
	}

	for id, sa := range *s {
		da, ok := dst[id]
		if !ok {
			return false
		}
		if (sa.Kind != da.Kind) ||
			(sa.Offset != da.Offset) ||
			!sa.ScheduleTime.Equal(da.ScheduleTime) ||
			(sa.Value != da.Value) ||
			!equalArgs(&sa.Args, &da.Args) {
			return false
		}
	}
	return true
}

// Clone method creates new state, copies all keys and state attributes to result
func (s *ActualState) Clone() ActualState {
	dst := MakeActualState()
	for id, a := range *s {
		dst[id] = a
	}
	return dst
}

// Equal returns is actual state equals to specified state `dst`
func (s *ActualState) Equal(dst ActualState) bool {
	if len(*s) != len(dst) {
		return false
	}

	for id, sa := range *s {
		da, ok := dst[id]
		if !ok {
			return false
		}
		if (sa.Kind != da.Kind) ||
			(sa.Offset != da.Offset) ||
			(sa.TimeMs != da.TimeMs) ||
			(sa.AttemptNum != da.AttemptNum) ||
			(sa.Status != da.Status) ||
			(sa.Error != da.Error) ||
			(sa.Info != da.Info) {
			return false
		}
	}
	return true
}

// Achieves() returns is actual state achieves specified desired state `dst`.
//
//	Ref ActualAttribute Achieves() method to see attribute achievement particulars
func (s *ActualState) Achieves(dst DesiredState) bool {
	for id, da := range dst {
		if da.Offset == 0 {
			continue
		}
		aa, ok := (*s)[id]
		if !ok {
			return false
		}
		if !aa.Achieves(da) {
			return false
		}
	}

	return true
}

// Achieves() returns is actual attribute achieves desired attribute `da`.
//
//	If desired attribute is empty (Offset is zero) then returns true.
//	If actual and desired offsets are different, then returns false.
//	If actual and desired offsets are equals and actual status is finished without error, then returns true.
func (aa ActualAttribute) Achieves(da DesiredAttribute) bool {
	if da.Offset == 0 {
		return true
	}
	if aa.Kind != da.Kind {
		return false
	}
	if aa.Offset != da.Offset {
		return false
	}
	if aa.Status != FinishedStatus {
		return false
	}
	if aa.Error != "" {
		return false
	}

	return true
}

type states struct{}

// IEdgeNodeState.GetDesiredState
func (s *states) GetDesiredState(ctx context.Context) (state DesiredState, err error) {
	//TODO: real getting
	return MakeDesiredState(), nil
}

// IEdgeNodeState.ReportActualState
func (s *states) ReportActualState(ctx context.Context, state ActualState) (err error) {
	//TODO: real reporting
	return nil
}

// Send method sends new state to channel. Send is immediately and synchrony.
// If Send calls repeatedly, it replaces previous state in channel.
func (lsc *LastStateChannel[T]) Send(state T) {
	select {
	case <-lsc.c: // this remove old state
	default:
	}

	lsc.c <- state
}

// ReceiveChannel method returns a receive-only channel to receive a last sent state.
func (lsc *LastStateChannel[T]) ReceiveChannel() <-chan T {
	return lsc.c
}
