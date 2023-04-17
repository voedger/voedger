/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package ctrls

import (
	"context"

	"github.com/untillpro/voedger/cmd/edger/internal/states"
)

// DefaultAchievedStateFileName is default file name used by SuperController to load and store last achieved state
const DefaultAchievedStateFileName = `edger-state.json`

// MockMicroControllerFactory constructs mocked microcontroller, which always successfully achieves desired state from first attempt
var MockMicroControllerFactory MicroControllerFactory = func() MicroController {
	return func(_ context.Context, desired states.DesiredAttribute) (status states.ActualStatus, info string, err error) {
		return states.FinishedStatus, "mocked microcontroller result", nil
	}
}
