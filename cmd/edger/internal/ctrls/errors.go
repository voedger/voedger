/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package ctrls

import "errors"

const (
	fmtCanNotFindRegisteredFactory = "can not find registered factory for state attribute %v: %w"
	fmtAchivingStateAttributeError = "can not achieve state attribute %v (ID = `%s`): %w"

	fmtReadingSuperControllerStateFileError = "error reading last actual state from file «%s»: %w"
	fmtWriteSuperControllerStateFileError   = "error writing last actual state to file «%s»: %w"
	fmtUnmarshalingStateFileError           = "error unmarshaling actual state loaded from file «%s»: %w"
	fmtMarshalingStateError                 = "error marshaling actual state: %w"
)

var (
	ErrNotFoundError = errors.New("not found error")
)
