/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package ctrls

import (
	"context"

	"github.com/untillpro/voedger/cmd/edger/internal/states"
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
