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

func GetAppWSID(wsid istructs.WSID, appWSAmount istructs.AppWSAmount) istructs.WSID {
	baseWSID := wsid.BaseWSID()
	appWSNumber := baseWSID % istructs.WSID(appWSAmount)
	baseAppWSID := istructs.FirstBaseAppWSID + appWSNumber
	return istructs.NewWSID(istructs.MainClusterID, baseAppWSID)
}

func CRC16(entity []byte) uint16 {
	return uint16(crc32.ChecksumIEEE(entity) & CRC16Mask)
}

func GetPseudoWSID(ownerWSID istructs.WSID, entity string, clusterID istructs.ClusterID) istructs.WSID {
	if ownerWSID != 0 {
		crc16 := CRC16([]byte(fmt.Sprint(ownerWSID) + "/" + entity))
		return istructs.NewWSID(clusterID, istructs.WSID(crc16))
	}
	crc16 := CRC16([]byte(entity))
	return istructs.NewWSID(clusterID, istructs.WSID(crc16))
}
