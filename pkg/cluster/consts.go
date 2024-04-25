/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import "embed"

//go:embed appws.vsql
var schemaFS embed.FS

const (
	ClusterPackage    = "cluster"
	ClusterPackageFQN = "github.com/voedger/voedger/pkg/" + ClusterPackage
)
