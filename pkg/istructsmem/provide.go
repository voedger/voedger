/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package istructsmem

import (
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
)

// Provide: constructs new application structures provider
func Provide(appConfigs AppConfigsType, appTokensFactory payloads.IAppTokensFactory,
	storageProvider istorage.IAppStorageProvider, seqTrustLevel isequencer.SequencesTrustLevel, appTTLStorageFactory istructs.AppTTLStorageFactory) (provider istructs.IAppStructsProvider) {
	return &appStructsProviderType{
		locker:               sync.RWMutex{},
		configs:              appConfigs,
		structures:           make(map[appdef.AppQName]*appStructsType),
		appTokensFactory:     appTokensFactory,
		storageProvider:      storageProvider,
		seqTrustLevel:        seqTrustLevel,
		appTTLStorageFactory: appTTLStorageFactory,
	}
}
