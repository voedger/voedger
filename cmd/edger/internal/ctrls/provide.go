/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package ctrls

import "github.com/voedger/voedger/cmd/edger/internal/states"

// New returns new ISuperController which uses specified microcontrollers factories and parameters
func New(factories map[states.AttributeKind]MicroControllerFactory, params SuperControllerParams) (ISuperController, error) {
	return newSuperController(factories, params)
}
