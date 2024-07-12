/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package iextenginewazero

import (
	"github.com/voedger/voedger/pkg/iextengine"
)

func ProvideExtensionEngineFactory(wasmConfig iextengine.WASMFactoryConfig) iextengine.IExtensionEngineFactory {
	return extensionEngineFactory{wasmConfig: wasmConfig}
}
