/*
 * Copyright (c) 2025-present unTill Pro, Ltd. and Contributors
 * @author Denis Gribanov
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package retrier

import "errors"

// ErrInvalidConfig is returned when Config has invalid values.
var ErrInvalidConfig = errors.New("invalid retry config")
