/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import "embed"

//go:embed schema.sql
var schemaSQL embed.FS

const (
	ClusterPackage = "cluster"
	ClusterAppFQN  = "github.com/voedger/voedger/pkg/apps/sys/clusterapp"
)
