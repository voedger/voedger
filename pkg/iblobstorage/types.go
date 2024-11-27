/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iblobstorage

import istructs "github.com/voedger/voedger/pkg/istructs"

type BLOBState struct {
	Descr      DescrType
	StartedAt  istructs.UnixMilli
	FinishedAt istructs.UnixMilli
	Size       uint64
	// Status must be above BLOBStatus_Unknown
	Status BLOBStatus
	// Not empty if error happened during upload
	Error string
	// 0 - the BLOB is persistent, otherwise - temporary
	Duration DurationType
}

type PersistentBLOBKeyType struct {
	AppID istructs.ClusterAppID
	WSID  istructs.WSID
	ID    istructs.RecordID
}

type TempBLOBKeyType struct {
	AppID istructs.ClusterAppID
	WSID  istructs.WSID
	SUUID SUUID
}

type SUUID string

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

type BLOBMaxSizeType uint64

type DurationType int

type WQuoterType func(wantToWriteBytes uint64) error

type RQuoterType func(wantReadBytes uint64) error
