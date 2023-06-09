/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package iblobstoragestg

import (
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istorage"
)

func Provide(storage istorage.IAppStorage, now coreutils.TimeFunc) iblobstorage.IBLOBStorage {
	return &bStorageType{
		appStorage: storage,
		now:        now,
	}
}
