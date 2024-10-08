/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package clusterapp

import (
	"embed"

	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

//go:embed schema.vsql
var schemaFS embed.FS

const (
	ClusterAppFQN           = "github.com/voedger/voedger/pkg/vvm/builtin/clusterapp"
	ClusterAppNumPartitions = istructs.NumAppPartitions(1)
	ClusterAppNumAppWS      = istructs.NumAppWorkspaces(1)
)

var (
	ClusterAppWSID            = istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID)    // 140737488420864
	ClusterAppPseudoWSID      = istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstPseudoBaseWSID) // 140737488355328
	ClusterAppWSIDPartitionID = coreutils.AppPartitionID(ClusterAppWSID, ClusterAppNumPartitions)           // 0
	ClusterAppNumEngines      = [appparts.ProcessorKind_Count]uint{uint(ClusterAppNumPartitions), 1, uint(ClusterAppNumPartitions)}
)
