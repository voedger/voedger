/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package ctrls

import (
	"context"

	"github.com/untillpro/voedger/cmd/edger/internal/states"
)

// MicroController is used by SuperController to achieve desired state attribute
type MicroController func(ctx context.Context, desired states.DesiredAttribute) (status states.ActualStatus, info string, err error)

// MicroControllerFactory is used by SuperController to construct new microcoontrollers for specified attribute kind
type MicroControllerFactory func() MicroController
