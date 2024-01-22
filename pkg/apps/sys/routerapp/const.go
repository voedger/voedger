/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package routerapp

import (
	"embed"

	"github.com/voedger/voedger/pkg/cluster"
)

//go:embed schema.sql
var routerAppSchemaFS embed.FS

const RouterAppFQN = "github.com/voedger/voedger/pkg/apps/sys/routerapp"

const DefDeploymentPartsCount = 10

var DefDeploymentEnginePoolSize = cluster.PoolSize(0, 0, 0)
