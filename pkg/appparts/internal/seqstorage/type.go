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

type implISeqStorage struct {
	partitionID isequencer.PartitionID
	events      istructs.IEvents
	seqIDs      map[appdef.QName]uint16
	storage     isequencer.ISeqSysVVMStorage
	appDef      appdef.IAppDef
}
