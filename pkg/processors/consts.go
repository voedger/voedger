/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package processors

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

const (
	Field_JSONDef_Body = "Body"
	fieldBodyLen       = appdef.MaxFieldLength
)

var ErrWSInactive = coreutils.NewHTTPErrorf(http.StatusGone, "workspace status is not active")
