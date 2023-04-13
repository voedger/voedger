/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iblobstorage

import istructs "github.com/untillpro/voedger/pkg/istructs"

type BLOBState struct {
	Descr      DescrType
	StartedAt  istructs.UnixMilli
	FinishedAt istructs.UnixMilli
	Size       int64
	// Status must be above BLOBStatus_Unknown
	Status BLOBStatus
	// Not empty if error happened during upload
	Error string
}

type KeyType struct {
	AppID istructs.ClusterAppID
	WSID  istructs.WSID
	ID    istructs.RecordID
}

type DescrType struct {
	Name     string
	MimeType string
}

type BLOBStatus uint8

const (
	BLOBStatus_Unknown BLOBStatus = iota
	BLOBStatus_InProcess
	BLOBStatus_Completed
)
