/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package uniques

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
)

// Uniques IDs system view.
//
//	Use ID() to obtain ID for unique.
//	Use Prepare() to load uniques from storage.
type Uniques struct {
	ids     map[string]appdef.UniqueID
	lastID  appdef.UniqueID
	changes uint
	qnames  *qnames.QNames
}
