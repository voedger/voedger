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

// MicroController is used by SuperController to achieve desired state attribute
type MicroController func(ctx context.Context, desired states.DesiredAttribute) (status states.ActualStatus, info string, err error)

// MicroControllerFactory is used by SuperController to construct new microcoontrollers for specified attribute kind
type MicroControllerFactory func() MicroController
