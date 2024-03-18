/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package iextengineimpl

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/iextenginebuiltin"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func ProvideExtEngineFactories(cfgs istructsmem.AppConfigsType) iextengine.ExtensionEngineFactories {
	return iextengine.ExtensionEngineFactories{
		appdef.ExtensionEngineKind_BuiltIn: iextenginebuiltin.ProvideExtensionEngineFactory(provideAppsBuiltInExtFuncs(cfgs)),
	}
}
