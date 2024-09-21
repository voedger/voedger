/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package processors

import (
	"net/http"

	"github.com/voedger/voedger/pkg/coreutils"
)

const Field_RawObject_Body = "Body"

var ErrWSInactive = coreutils.NewHTTPErrorf(http.StatusGone, "workspace status is not active")
