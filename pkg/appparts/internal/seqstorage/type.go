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

// [~server.design.sequences/cmp.ISeqStorageImplementation~impl]
type implISeqStorage struct {
	appID       isequencer.ClusterAppID
	partitionID istructs.PartitionID
	events      istructs.IEvents
	seqIDs      map[appdef.QName]istructs.QNameID
	storage     isequencer.IVVMSeqStorageAdapter
	appDef      appdef.IAppDef
}
