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
)

type ExtEngineFactoriesConfig struct {
	AppConfig   *istructsmem.AppConfigType
	WASMCompile bool
}

func ProvideExtEngineFactories(cfg ExtEngineFactoriesConfig) iextengine.ExtensionEngineFactories {
	return iextengine.ExtensionEngineFactories{
		appdef.ExtensionEngineKind_BuiltIn: builtin.ProvideExtensionEngineFactory(provideAppsBuiltInExtFuncs(cfg.AppConfig)),
		appdef.ExtensionEngineKind_WASM:    wazero.ProvideExtensionEngineFactory(cfg.WASMCompile),
	}
}
