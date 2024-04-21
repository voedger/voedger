/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package btstrp

import (
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

const clusterAppNumPartitions = istructs.NumAppPartitions(1)

var (
	clusterAppWSID            = istructs.NewWSID(istructs.MainClusterID, istructs.FirstBaseAppWSID) // 140737488420864
	clusterAppWSIDPartitionID = coreutils.AppPartitionID(clusterAppWSID, clusterAppNumPartitions) // 0
	clusterAppNumEngines      = [cluster.ProcessorKind_Count]int{int(clusterAppNumPartitions), 1, int(clusterAppNumPartitions)}
)
