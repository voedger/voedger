/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package istoragecache

import (
	istorage "github.com/voedger/voedger/pkg/istorage"
	imetrics "github.com/voedger/voedger/pkg/metrics"
)

// Provide s.e.
func Provide(maxBytes int, storageInitializer istorage.IAppStorageInitializer, metrics imetrics.IMetrics, vvmName string) istorage.IAppStorageInitializer {
	return &implCachingAppStorageInitializer{
		maxBytes:           maxBytes,
		storageInitializer: storageInitializer,
		metrics:            metrics,
		vvmName:            vvmName,
	}
}
