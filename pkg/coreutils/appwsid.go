/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"fmt"
	"hash/crc32"

	"github.com/voedger/voedger/pkg/istructs"
)

func GetAppWSID(wsid istructs.WSID, numAppWorkspaces istructs.NumAppWorkspaces) istructs.WSID {
	baseWSID := wsid.BaseWSID()
	appWSNumber := baseWSID % istructs.WSID(numAppWorkspaces)
	baseAppWSID := istructs.FirstBaseAppWSID + appWSNumber
	// problem: app workspaces are automatically created in the main cluster on VVM launch
	// request to an another cluster -> there are no App Workspaces yet
	// it is ok for now because App Workspaces should be created on deply an app in the new cluster
	// we're using Main Cluster only
	return istructs.NewWSID(istructs.CurrentClusterID(), baseAppWSID)
}

func CRC16(entity []byte) uint16 {
	return uint16(crc32.ChecksumIEEE(entity) & CRC16Mask) //nolint G115
}

// for Login use NullWSID as ownerWSID
func GetPseudoWSID(ownerWSID istructs.WSID, entity string, clusterID istructs.ClusterID) istructs.WSID {
	if ownerWSID != 0 {
		entity = fmt.Sprint(ownerWSID) + "/" + entity
	}
	crc16 := CRC16([]byte(entity))
	return istructs.NewWSID(clusterID, istructs.WSID(crc16))
}

// resulting pseudoWSID leads to the initial appWSID
// note: there could be many different pseudoWSIDs that leads to the same appWSID
func AppWSIDToPseudoWSID(appWSID istructs.WSID) (pseudoWSID istructs.WSID) {
	appWSNumber := appWSID.BaseWSID() - istructs.FirstBaseAppWSID
	return istructs.NewWSID(istructs.CurrentClusterID(), appWSNumber)
}

// used in BuildAppWorkspaces() only because there are no apps in IAppPartitions on that moment
func AppPartitionID(wsid istructs.WSID, numAppPartitions istructs.NumAppPartitions) istructs.PartitionID {
	// numAppPartitions is uint16, the modulo operation is always less than numAppPartitions
	// so data loss is not possible on casting to PartitionID
	return istructs.PartitionID(wsid % istructs.WSID(numAppPartitions)) // nolint G115
}
