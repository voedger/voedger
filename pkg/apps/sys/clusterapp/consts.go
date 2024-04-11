/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package clusterapp

import "embed"

//go:embed schema.vsql
var schemaFS embed.FS

const ClusterAppFQN = "github.com/voedger/voedger/pkg/apps/sys/clusterapp"
