/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package blobberapp

import (
	"embed"

	"github.com/voedger/voedger/pkg/cluster"
)

//go:embed schema.sql
var blobberSchemaFS embed.FS

const BlobberAppFQN = "github.com/voedger/voedger/pkg/apps/sys/blobberapp"

const DefDeploymentPartsCount = 10

var DefDeploymentEnginePoolSize = cluster.PoolSize(10, 10, 0)
