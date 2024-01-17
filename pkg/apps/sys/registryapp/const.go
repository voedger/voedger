/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package registryapp

import (
	"embed"

	"github.com/voedger/voedger/pkg/cluster"
)

//go:embed schema.sql
var registryAppSchemaFS embed.FS

const RegistryAppFQN = "github.com/voedger/voedger/pkg/apps/sys/registryapp"

const DefDeploymentPartsCount = 10

var DefDeploymentEnginePoolSize = cluster.PoolSize(10, 10, 10)
