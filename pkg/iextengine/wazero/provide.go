/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package iextenginewazero

import (
	"github.com/voedger/voedger/pkg/iextengine"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/processors"
)

func ProvideExtensionEngineFactory(wasmConfig iextengine.WASMFactoryConfig, vvmName processors.VVMName, imetrics imetrics.IMetrics) iextengine.IExtensionEngineFactory {
	return extensionEngineFactory{wasmConfig: wasmConfig, vvmName: vvmName, imetrics: imetrics}
}
