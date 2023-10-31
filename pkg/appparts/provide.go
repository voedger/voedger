/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import "github.com/voedger/voedger/pkg/istorage"

func New(storages istorage.IAppStorageProvider) (ap IAppPartitions, cleanup func(), err error) {
	return newAppPartitions(storages)
}
