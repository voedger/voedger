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

type FilledAppConfigsType struct{ istructsmem.AppConfigsType }

func ProvideExtEngineFactories(cfgs FilledAppConfigsType) iextengine.ExtensionEngineFactories {
	return iextengine.ExtensionEngineFactories{
		appdef.ExtensionEngineKind_BuiltIn: iextenginebuiltin.ProvideExtensionEngineFactory(provideAppsBuiltInExtFuncs(cfgs)),
	}
}
