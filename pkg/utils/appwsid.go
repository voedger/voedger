/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import "github.com/voedger/voedger/pkg/istructs"

func GetAppWSID(wsid istructs.WSID, appWSAmount istructs.AppWSAmount) istructs.WSID {
	baseWSID := wsid.BaseWSID()
	appWSNumber := baseWSID % istructs.WSID(appWSAmount)
	baseAppWSID := istructs.FirstBaseAppWSID + appWSNumber
	return istructs.NewWSID(istructs.MainClusterID, baseAppWSID)
}
