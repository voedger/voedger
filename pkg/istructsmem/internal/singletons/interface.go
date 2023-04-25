/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package singletons

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

// Singletons system view
type Singletons struct {
	qNames  map[schemas.QName]istructs.RecordID
	ids     map[istructs.RecordID]schemas.QName
	lastID  istructs.RecordID
	changes uint
}
