/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package iblobstoragestg

import (
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istorage"
)

type BlobAppStoragePtr *istorage.IAppStorage

type implSizeLimiter struct {
	uploadedSize uint64
	maxSize      iblobstorage.BLOBMaxSizeType
}
