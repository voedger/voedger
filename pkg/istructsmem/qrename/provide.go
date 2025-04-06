/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qrename

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
)

// Renames QName from old to new. QNameID previously used by old will be used by new.
func Rename(storage istorage.IAppStorage, oldQName, newQName appdef.QName) error {
	return qnames.Rename(storage, oldQName, newQName)
}
