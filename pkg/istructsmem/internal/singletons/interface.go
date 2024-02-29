/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package singletons

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// Singletons IDs system view.
//
//	Use ID() to obtain singleton ID by its QName.
//	Use Prepare() to load Singletons from storage.
type Singletons struct {
	qNames  map[appdef.QName]istructs.RecordID
	ids     map[istructs.RecordID]appdef.QName
	lastID  istructs.RecordID
	changes uint
}
