/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package ihttpctl

import (
	"io/fs"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

type StaticResourcesType map[string]fs.FS
type RedirectRoutes map[string]string
type DefaultRedirectRoute map[string]string // single record only
type AppRequestHandler struct {
	AppQName      appdef.AppQName
	NumPartitions uint
	NumAppWS      uint
	Handlers      map[istructs.PartitionID]ibus.RequestHandler
}
type AppRequestHandlers []AppRequestHandler
