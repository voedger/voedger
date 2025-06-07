/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package iblobstoragestg

import "github.com/voedger/voedger/pkg/iblobstorage"

const (
	chunkSize  uint64 = 102400
	bucketSize uint64 = 100
)

var (
	RLimiter_Null iblobstorage.RLimiterType = func(wantReadBytes uint64) error { return nil }
	cColState                               = []byte{0, 0, 0, 0, 0, 0, 0, 0}
)
