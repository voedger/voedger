/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package iblobstorage

import (
	"io"

	"github.com/voedger/voedger/pkg/appdef"
	istructs "github.com/voedger/voedger/pkg/istructs"
)

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
	// TODO: it is not a state, it is like properties. Descr - wrong because we provide descr to WriteBLOB and possible: duration>0 but PersistentBLOBKey is provided
	Duration DurationType
}

type PersistentBLOBKeyType struct {
	ClusterAppID istructs.ClusterAppID
	WSID         istructs.WSID
	BlobID       istructs.RecordID
}

type TempBLOBKeyType struct {
	ClusterAppID istructs.ClusterAppID
	WSID         istructs.WSID
	SUUID        SUUID
}

type SUUID string

type DescrType struct {
	Name        string
	ContentType string

	// empty for temp blobs
	OwnerRecord      appdef.QName
	OwnerRecordField appdef.FieldName
}

type BLOBStatus uint8

const (
	BLOBStatus_Unknown BLOBStatus = iota
	BLOBStatus_InProcess
	BLOBStatus_Completed
)

type BLOBMaxSizeType uint64

// DurationType - amount of days to store the BLOB
type DurationType int

type WLimiterType func(wantToWriteBytes uint64) error

type RLimiterType func(wantReadBytes uint64) error

type blobPrefix uint64

// for read and write
// caller must read out and close the reader
type BLOBReader struct {
	io.ReadCloser
	DescrType
}
