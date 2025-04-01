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
		events:  events,
		appId:   AppID,
		appDef:  appDef,
		storage: seqStorage,
		seqIDs: map[appdef.QName]uint16{
			istructs.QNameWLogOffsetSequence: istructs.QNameIDWLogOffsetSequence,
			istructs.QNameCRecordIDSequence:  istructs.QNameIDCRecordIDSequence,
			istructs.QNameOWRecordIDSequence: istructs.QNameIDOWRecordIDSequence,
		},
	}
}
