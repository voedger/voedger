/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"github.com/voedger/voedger/pkg/istructs"
)

func New(structs istructs.IAppStructsProvider) (ap IAppPartitions, cleanup func(), err error) {
	return newAppPartitions(structs)
}
