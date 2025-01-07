/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package ihttpctl

import (
	"io/fs"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/istructs"
)

type StaticResourcesType map[string]fs.FS
type RedirectRoutes map[string]string
type DefaultRedirectRoute map[string]string // single record only
type AppRequestHandler struct {
	AppQName      appdef.AppQName
	NumPartitions istructs.NumAppPartitions
	NumAppWS      istructs.NumAppWorkspaces
	Handlers      map[istructs.PartitionID]bus.RequestHandler
}
type AppRequestHandlers []AppRequestHandler
