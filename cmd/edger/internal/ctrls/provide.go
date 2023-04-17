/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package ctrls

import "github.com/untillpro/voedger/cmd/edger/internal/states"

// New returns new ISuperController which uses specified microcontrollers factories and parameters
func New(factories map[states.AttributeKind]MicroControllerFactory, params SuperControllerParams) ISuperController {
	return newSuperController(factories, params)
}
