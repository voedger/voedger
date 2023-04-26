/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qrename

import (
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/schemas"
)

func RenameQName(storage istorage.IAppStorage, old, new schemas.QName) error {
	return qnames.Rename(storage, old, new)
}
