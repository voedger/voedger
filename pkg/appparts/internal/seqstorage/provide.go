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

// [~server.design.sequences/cmp.ISeqStorageImplementation.New~impl]
func New(appID isequencer.ClusterAppID, partitionID istructs.PartitionID, events istructs.IEvents, appDef appdef.IAppDef,
	seqStorageAdapter isequencer.IVVMSeqStorageAdapter) isequencer.ISeqStorage {
	return &implISeqStorage{
		events:      events,
		partitionID: partitionID,
		appID:       appID,
		appDef:      appDef,
		storage:     seqStorageAdapter,
		seqIDs: map[appdef.QName]uint16{
			istructs.QNameWLogOffsetSequence: istructs.QNameIDWLogOffsetSequence,
			istructs.QNameRecordIDSequence:   istructs.QNameIDRecordIDSequence,
		},
	}
}
