/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package istoragecache

import (
	istorage "github.com/untillpro/voedger/pkg/istorage"
	imetrics "github.com/untillpro/voedger/pkg/metrics"
)

// Provide s.e.
func Provide(maxBytes int, storageProvider istorage.IAppStorageProvider, metrics imetrics.IMetrics, hvmName string) istorage.IAppStorageProvider {
	return &implCachingAppStorageProvider{
		maxBytes:        maxBytes,
		storageProvider: storageProvider,
		metrics:         metrics,
		hvmName:         hvmName,
	}
}
