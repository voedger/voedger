/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package states

import (
	"context"
	"time"
)

// IEdgeNodeState provides receiving desired state of edge node from Heeus cloud and reporting actual state to Heeus cloud
type IEdgeNodeState interface {
	// GetDesiredState receives desired state of edge node from Heeus cloud.
	// Returned `state` value must be unique for each return and can not be reused, as will be sent to a channel to work in another go-routine.
	GetDesiredState(ctx context.Context) (state DesiredState, err error)
	// ReportActualState reports actual state to Heeus cloud
	ReportActualState(ctx context.Context, state ActualState) (err error)
}

// AttributeKind is enumeration of available attribute value (docker stack, edger binaries, command)
type AttributeKind int8

// AttrOffset is incremental version of attribute
type AttrOffset uint64

// DesiredState describes the desired state of edge node. Key is attribute ID.
type DesiredState map[string]DesiredAttribute

// DesiredAttribute describes the desired attribute state (docker stacks, edger binaries, command)
type DesiredAttribute struct {
	Kind         AttributeKind
	Offset       AttrOffset
	ScheduleTime time.Time
	Value        string
	Args         []string
}

// ActualStatus is enumeration of available status (pending, in progress, finished)
type ActualStatus int8

// ActualState describes the desired state of edge node. Key is attribute ID
type ActualState map[string]ActualAttribute

// ActualAttribute describes the actual attribute state (docker stack or edger or command)
type ActualAttribute struct {
	Kind AttributeKind
	// Zero if attribute not assigned
	Offset AttrOffset
	// time.Now().UnixMilli must be used to set
	TimeMs     int64
	AttemptNum uint
	Status     ActualStatus

	// if Error is empty and Status is finished, then success
	Error string
	Info  string
}

// LastStateChannel is a special state channel. Channel contains zero or one state — last received state.
// Use Send() method to send new state.
// Use ReceiveChannel() method to obtain channel to receive last state.
type LastStateChannel[T DesiredState | ActualState] struct {
	c chan T
}
