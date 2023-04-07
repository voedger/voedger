/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package istructsmem

import (
	"sync"

	"github.com/untillpro/voedger/pkg/irates"
	"github.com/untillpro/voedger/pkg/istorage"
	"github.com/untillpro/voedger/pkg/istructs"
	payloads "github.com/untillpro/voedger/pkg/itokens-payloads"
)

// Provide: constructs new application structures provider
func Provide(appConfigs AppConfigsType, bucketsFactory irates.BucketsFactoryType, appTokensFactory payloads.IAppTokensFactory,
	storageProvider istorage.IAppStorageProvider) (provider istructs.IAppStructsProvider, err error) {
	return &appStructsProviderType{
		locker:           sync.RWMutex{},
		configs:          appConfigs,
		structures:       make(map[istructs.AppQName]*appStructsType),
		bucketsFacotry:   bucketsFactory,
		appTokensFactory: appTokensFactory,
		storageProvider:  storageProvider,
	}, nil
}
