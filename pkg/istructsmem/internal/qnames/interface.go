/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"github.com/voedger/voedger/pkg/schemas"
)

// Identificator for QNames
type QNameID uint16

// QNames system view
type QNames struct {
	qNames  map[schemas.QName]QNameID
	ids     map[QNameID]schemas.QName
	lastID  QNameID
	changes uint
}
