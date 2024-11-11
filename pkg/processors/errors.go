/*
 * Copyright (c) 2024-present unTill Pro, Ltd. and Contributors
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package processors

import (
	"net/http"

	"github.com/voedger/voedger/pkg/coreutils"
)

var (
	ErrWSNotInited = coreutils.NewHTTPErrorf(http.StatusForbidden, "workspace is not initialized")
	ErrWSInactive  = coreutils.NewHTTPErrorf(http.StatusGone, "workspace status is not active")
)
