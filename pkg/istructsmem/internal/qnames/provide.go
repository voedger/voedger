/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/schemas"
)

// Create and return new QNames
func NewQNames() *QNames {
	return newQNames()
}

// Renames QName from old to new. QNameID previously used by old will be used by new.
func RenameQName(storage istorage.IAppStorage, old, new schemas.QName) error {
	return renameQName(storage, old, new)
}
