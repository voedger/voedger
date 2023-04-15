/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package iblobstoragestg

import (
	"time"

	"github.com/untillpro/voedger/pkg/iblobstorage"
	"github.com/untillpro/voedger/pkg/istorage"
)

func Provide(storage istorage.IAppStorage, now func() time.Time) iblobstorage.IBLOBStorage {
	return &bStorageType{
		appStorage: storage,
		now:        now,
	}
}
