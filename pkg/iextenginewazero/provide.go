/*
  - Copyright (c) 2023-present unTill Software Development Group B. V.
    @author Michael Saigachenko
*/
package iextenginewasm

import "github.com/voedger/voedger/pkg/iextengine"

func ProvideExtensionEngineFactory(compile bool) iextengine.IExtensionEngineFactory {
	return extensionEngineFactory{compile}
}
