/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package ctrls

import (
	"context"

	"github.com/voedger/voedger/cmd/edger/internal/states"
)

// ISuperController
type ISuperController interface {
	// AchieveState achieves desired node state.
	// Returned `achieved` state must be unique for each return and can not be reused, as will be sent to a channel to work in another go-routine.
	AchieveState(ctx context.Context, desired states.DesiredState) (achieved states.ActualState, err error)
}

// SuperControllerParams
type SuperControllerParams struct {
	// AchievedStateFile is full path file name to load and store last achieved state.
	// If omitted, then `edger-state.json` file in current working directory is used.
	AchievedStateFile string
}
