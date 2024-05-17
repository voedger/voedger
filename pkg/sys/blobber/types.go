/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobber

import (
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type StoredBLOB struct {
	coreutils.BLOB
	RecordID istructs.RecordID
}
