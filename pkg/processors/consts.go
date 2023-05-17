/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package processors

import (
	"net/http"

	coreutils "github.com/voedger/voedger/pkg/utils"
)

const Field_JSONDef_Body = "Body"

var ErrWSInactive = coreutils.NewHTTPErrorf(http.StatusForbidden, "workspace status is not active")
