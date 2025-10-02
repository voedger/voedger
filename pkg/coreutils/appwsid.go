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

func PseudoWSIDToAppWSID(wsid istructs.WSID, numAppWorkspaces istructs.NumAppWorkspaces) istructs.WSID {
	baseAppWSID := istructs.FirstBaseAppWSID + istructs.WSID(AppWSNumber(wsid, numAppWorkspaces))
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

// used in BuildAppWorkspaces() only because there are no apps in IAppPartitions on that moment
func AppPartitionID(wsid istructs.WSID, numAppPartitions istructs.NumAppPartitions) istructs.PartitionID {
	// numAppPartitions is uint16, the modulo operation is always less than numAppPartitions
	// so data loss is not possible on casting to PartitionID
	return istructs.PartitionID(wsid % istructs.WSID(numAppPartitions)) // nolint G115
}

func AppWSNumber(appWSID istructs.WSID, numAppWorkspaces istructs.NumAppWorkspaces) uint32 {
	// no data loss possible because numAppWorkspaces are expected to be less than istructs.MaxAppWorkspaces(32768)
	return uint32(appWSID.BaseWSID() % istructs.WSID(numAppWorkspaces)) // nolint G115
}
