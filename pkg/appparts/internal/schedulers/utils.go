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
func AppWorkspacesHandledByPartition(numAppPartitions istructs.NumAppPartitions, numAppWorkspaces istructs.NumAppWorkspaces, part istructs.PartitionID) map[istructs.WSID]istructs.AppWorkspaceNumber {
	appWSNumbers := make(map[istructs.WSID]istructs.AppWorkspaceNumber)
	for appWorspaceIdx := istructs.NumAppWorkspaces(0); appWorspaceIdx < numAppWorkspaces; appWorspaceIdx++ {
		appWSID := istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(appWorspaceIdx)+istructs.FirstBaseAppWSID)
		if coreutils.AppPartitionID(appWSID, numAppPartitions) == part {
			appWSNumbers[appWSID] = istructs.AppWorkspaceNumber(appWorspaceIdx)
		}
	}
	return appWSNumbers
}
