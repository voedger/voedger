/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package istoragecache

import (
	istorage "github.com/voedger/voedger/pkg/istorage"
	imetrics "github.com/voedger/voedger/pkg/metrics"
)

// Provide s.e.
func Provide(maxBytes int, storageProvider istorage.IAppStorageProvider, metrics imetrics.IMetrics, vvmName string) istorage.IAppStorageProvider {
	return &implCachingAppStorageProvider{
		maxBytes:        maxBytes,
		storageProvider: storageProvider,
		metrics:         metrics,
		vvmName:         vvmName,
	}
}
