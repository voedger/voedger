/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package blobberapp

import (
	"embed"

	"github.com/voedger/voedger/pkg/appparts"
)

//go:embed schema.vsql
var blobberSchemaFS embed.FS

const BlobberAppFQN = "github.com/voedger/voedger/pkg/vvm/builtin/blobberapp"

const DefDeploymentPartsCount = 10

var DefDeploymentEnginePoolSize = appparts.PoolSize(10, 10, 10, 1)
