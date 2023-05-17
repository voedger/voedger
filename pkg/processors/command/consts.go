/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

const (
	intentsLimit = 128
)

var (
	ViewQNamePLogKnownOffsets = appdef.NewQName(appdef.SysPackage, "PLogKnownOffsets")
	ViewQNameWLogKnownOffsets = appdef.NewQName(appdef.SysPackage, "WLogKnownOffsets")
	errWSNotInited            = coreutils.NewHTTPErrorf(http.StatusForbidden, "workspace is not initialized")
	errWSInactive             = coreutils.NewHTTPErrorf(http.StatusForbidden, "workspace status is not active")
)
