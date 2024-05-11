/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package iblobstoragestg

import (
	"github.com/voedger/voedger/pkg/iblobstorage"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(storage BlobAppStoragePtr, now coreutils.TimeFunc) iblobstorage.IBLOBStorage {
	return &bStorageType{
		appStorage: storage,
		now:        now,
	}
}
