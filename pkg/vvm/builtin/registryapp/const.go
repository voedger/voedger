/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package registryapp

import (
	"embed"
)

//go:embed schema.vsql
var registryAppSchemaFS embed.FS

const RegistryAppFQN = "github.com/voedger/voedger/pkg/vvm/builtin/registryapp"

const (
	// query processors
	DefDeploymentQPCount = 10
	// scheduler processors
	DefDeploymentSPCount = 1
)
