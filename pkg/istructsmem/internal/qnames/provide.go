/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
)

// Create and return new QNames
func New() *QNames {
	return newQNames()
}

// Renames QName from old to new. QNameID previously used by old will be used by new.
func Rename(storage istorage.IAppStorage, oldQName, newQName appdef.QName) error {
	return renameQName(storage, oldQName, newQName)
}
