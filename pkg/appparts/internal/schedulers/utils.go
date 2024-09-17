/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schedulers

import (
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

// Returns application workspaces handled by the specified partition.
//
// Returned map keys - workspace IDs, values - workspace indexes in application workspaces list.
func AppWorkspacesHandledByPartition(numAppPartitions istructs.NumAppPartitions, numAppWorkspaces istructs.NumAppWorkspaces, part istructs.PartitionID) map[istructs.WSID]int {
	appWSNumbers := make(map[istructs.WSID]int)
	for appWorspaceIdx := 0; appWorspaceIdx < int(numAppWorkspaces); appWorspaceIdx++ {
		appWSID := istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(appWorspaceIdx+int(istructs.FirstBaseAppWSID)))
		if coreutils.AppPartitionID(appWSID, numAppPartitions) == part {
			appWSNumbers[appWSID] = appWorspaceIdx
		}
	}
	return appWSNumbers
}
