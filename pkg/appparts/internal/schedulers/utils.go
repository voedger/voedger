/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schedulers

import "github.com/voedger/voedger/pkg/istructs"

// Returns application workspaces handled by the specified partition.
//
// Returned map keys - workspace IDs, values - workspace indexes in application workspaces list.
func AppWorkspacesHandledByPartition(partCount istructs.NumAppPartitions, wsCount istructs.NumAppWorkspaces, part istructs.PartitionID) map[istructs.WSID]int {
	ws := make(map[istructs.WSID]int)
	for wsNum := 0; wsNum < int(wsCount); wsNum++ {
		wsID := istructs.NewWSID(istructs.MainClusterID, istructs.WSID(wsNum+int(istructs.FirstBaseAppWSID)))
		if int64(wsID)%int64(partCount) == int64(part) {
			ws[wsID] = wsNum
		}
	}
	return ws
}
