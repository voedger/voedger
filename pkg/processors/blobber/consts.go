/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import "github.com/voedger/voedger/pkg/iblobstorage"

const (
	temporaryBLOBIDLenTreshold = 40 // greater -> temporary, persistent oherwise
	branchReadBLOB             = "readBLOB"
	branchWriteBLOB            = "writeBLOB"
)

var (
	durationToRegisterFuncs = map[iblobstorage.DurationType]string{
		iblobstorage.DurationType_1Day: "c.sys.RegisterTempBLOB1d",
	}
)
