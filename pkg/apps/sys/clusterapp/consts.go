/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package clusterapp

import (
	"embed"

	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

//go:embed schema.vsql
var schemaFS embed.FS

const (
	ClusterAppFQN           = "github.com/voedger/voedger/pkg/apps/sys/clusterapp"
	ClusterAppNumPartitions = istructs.NumAppPartitions(1)
)

var (
	ClusterAppWSID            = istructs.NewWSID(istructs.MainClusterID, istructs.FirstBaseAppWSID) // 140737488420864
	ClusterAppWSIDPartitionID = coreutils.AppPartitionID(ClusterAppWSID, ClusterAppNumPartitions)   // 0
	ClusterAppNumEngines      = [cluster.ProcessorKind_Count]int{int(ClusterAppNumPartitions), 1, int(ClusterAppNumPartitions)}
)
