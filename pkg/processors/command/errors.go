/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package commandprocessor

import (
	"net/http"

	"github.com/voedger/voedger/pkg/sys/builtin"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

var ErrTooManyCUDs = coreutils.NewHTTPErrorf(http.StatusBadRequest, "too many cuds, max is", builtin.MaxCUDs)
