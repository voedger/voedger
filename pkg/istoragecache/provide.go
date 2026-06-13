/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package istoragecache

import (
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/metrics"
)

// Provide s.e.
func Provide(
	maxBytes int,
	storageProvider istorage.IAppStorageProvider,
	metrics imetrics.IMetrics,
	vvmName string,
	iTime timeu.ITime,
) istorage.IAppStorageProvider {
	return &implCachingAppStorageProvider{
		maxBytes:        maxBytes,
		storageProvider: storageProvider,
		metrics:         metrics,
		vvmName:         vvmName,
		iTime:           iTime,
	}
}
