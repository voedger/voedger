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

func New(appID istructs.ClusterAppID, partitionID istructs.PartitionID, events istructs.IEvents, appDef appdef.IAppDef,
	seqStorage isequencer.IVVMSeqStorageAdapter) isequencer.ISeqStorage_new {
	return &implISeqStorage{
		events:      events,
		partitionID: partitionID,
		appID:       appID,
		appDef:      appDef,
		storage:     seqStorage,
		seqIDs: map[appdef.QName]uint16{
			istructs.QNamePLogOffsetSequence: istructs.QNameIDPLogOffsetSequence,
			istructs.QNameWLogOffsetSequence: istructs.QNameIDWLogOffsetSequence,
			istructs.QNameCRecordIDSequence:  istructs.QNameIDCRecordIDSequence,
			istructs.QNameOWRecordIDSequence: istructs.QNameIDOWRecordIDSequence,
		},
	}
}
