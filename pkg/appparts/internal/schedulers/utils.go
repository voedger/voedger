/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schedulers

import (
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// Returns application workspaces handled by the specified partition.
//
// Returned map keys - workspace IDs, values - workspace indexes in application workspaces list.
func AppWorkspacesHandledByPartition(partCount istructs.NumAppPartitions, wsCount istructs.NumAppWorkspaces, part istructs.PartitionID) map[istructs.WSID]int {
	ws := make(map[istructs.WSID]int)
	for wsNum := 0; wsNum < int(wsCount); wsNum++ {
		wsID := istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(wsNum+int(istructs.FirstBaseAppWSID)))
		if coreutils.AppPartitionID(wsID, partCount) == part {
			ws[wsID] = wsNum
		}
	}
	return ws
}
