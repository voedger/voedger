/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package iblobstoragestg

import (
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iblobstorage"
)

func Provide(storage BlobAppStoragePtr, time coreutils.ITime) iblobstorage.IBLOBStorage {
	return &bStorageType{
		appStorage: storage,
		time:       time,
	}
}
