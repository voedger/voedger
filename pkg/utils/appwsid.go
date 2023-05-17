/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"hash/crc32"

	"github.com/voedger/voedger/pkg/istructs"
)

func GetAppWSID(wsid istructs.WSID, appWSAmount istructs.AppWSAmount) istructs.WSID {
	baseWSID := wsid.BaseWSID()
	appWSNumber := baseWSID % istructs.WSID(appWSAmount)
	baseAppWSID := istructs.FirstBaseAppWSID + appWSNumber
	// problem: app workspaces are automatically created in the main cluster on VVM launch
	// request to an another cluster -> there are no App Workspaces yet
	// it is ok for now because App Workspaces should be created on deply an app in the new cluster
	// we're using Main Cluster only
	return istructs.NewWSID(istructs.MainClusterID, baseAppWSID)
}

func CRC16(entity []byte) uint16 {
	return uint16(crc32.ChecksumIEEE(entity) & CRC16Mask)
}

func GetPseudoWSID(entity string, clusterID istructs.ClusterID) istructs.WSID {
	crc16 := CRC16([]byte(entity))
	return istructs.NewWSID(clusterID, istructs.WSID(crc16))
}
