/*
  - Copyright (c) 2023-present unTill Software Development Group B. V.
    @author Michael Saigachenko
*/
package iextengine

import "github.com/voedger/voedger/pkg/appdef"

type extensionEngineFactories struct{}

func (e extensionEngineFactories) QueryFactory(kind appdef.ExtensionEngineKind) IExtensionEngineFactory {
	switch kind {
	case appdef.ExtensionEngineKind_BuiltIn:
		return nil // TODO: implement
	case appdef.ExtensionEngineKind_WASM:
		return nil // TODO: implement
	}
	panic("undefined extension engine kind")
}
