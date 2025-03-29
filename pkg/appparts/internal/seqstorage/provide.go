/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package seqstorage

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istructs"
)

func New(partitionID istructs.PartitionID, events istructs.IEvents, appDef appdef.IAppDef,
	seqStorage isequencer.ISeqSysVVMStorage) isequencer.ISeqStorage {
	return &implISeqStorage{
		events:      events,
		partitionID: isequencer.PartitionID(partitionID),
		appDef:      appDef,
		storage:     seqStorage,
	}
}
