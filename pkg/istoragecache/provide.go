/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package istoragecache

import (
	"github.com/voedger/voedger/pkg/istorage"
	imetrics "github.com/voedger/voedger/pkg/metrics"
)

// Provide s.e.
func Provide(conf StorageCacheConf, storageProvider istorage.IAppStorageProvider, metrics imetrics.IMetrics, vvmName string) istorage.IAppStorageProvider {
	return &implCachingAppStorageProvider{
		conf:            conf,
		storageProvider: storageProvider,
		metrics:         metrics,
		vvmName:         vvmName,
	}
}
