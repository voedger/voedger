/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package routerapp

import (
	"embed"

	"github.com/voedger/voedger/pkg/appparts"
)

//go:embed schema.vsql
var routerAppSchemaFS embed.FS

const RouterAppFQN = "github.com/voedger/voedger/pkg/vvm/builtin/routerapp"

const DefDeploymentPartsCount = 10

var DefDeploymentEnginePoolSize = appparts.PoolSize(0, 0, 10, 1)
