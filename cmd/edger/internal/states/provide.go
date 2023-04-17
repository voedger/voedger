/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package states

import "time"

// New returns newly constructed IEdgeNodeState interface
func New() IEdgeNodeState {
	return &states{}
}

// MakeDesiredState makes a new DesiredState
func MakeDesiredState() DesiredState {
	return DesiredState{}
}

// MakeActualState makes a new ActualState
func MakeActualState() ActualState {
	return ActualState{}
}

// NewDesiredStateChannel returns newly constructed LastStateChannel for specified state (desired or actual)
func NewLastStateChannel[T DesiredState | ActualState]() *LastStateChannel[T] {
	ch := LastStateChannel[T]{c: make(chan T, 1)}
	return &ch
}

// Has the scheduled time arrived?
func IsScheduledTimeArrived(scheduledTime time.Time) bool {
	return isScheduledTimeArrived(scheduledTime, time.Now())
}
