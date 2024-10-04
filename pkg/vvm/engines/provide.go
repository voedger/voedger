/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package engines

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iextengine"
	builtin "github.com/voedger/voedger/pkg/iextengine/builtin"
	wazero "github.com/voedger/voedger/pkg/iextengine/wazero"
	"github.com/voedger/voedger/pkg/istructsmem"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/processors"
)

type ExtEngineFactoriesConfig struct {
	StatelessResources istructsmem.IStatelessResources
	AppConfigs         istructsmem.AppConfigsType
	WASMConfig         iextengine.WASMFactoryConfig
}

func ProvideExtEngineFactories(cfg ExtEngineFactoriesConfig, vvmName processors.VVMName, imetrics imetrics.IMetrics) iextengine.ExtensionEngineFactories {
	return iextengine.ExtensionEngineFactories{
		appdef.ExtensionEngineKind_BuiltIn: builtin.ProvideExtensionEngineFactory(
			provideAppsBuiltInExtFuncs(cfg.AppConfigs),
			provideStatelessFuncs(cfg.StatelessResources)),
		appdef.ExtensionEngineKind_WASM: wazero.ProvideExtensionEngineFactory(cfg.WASMConfig, vvmName, imetrics),
	}
}
