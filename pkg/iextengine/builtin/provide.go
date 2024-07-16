/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package iextenginebuiltin

import "github.com/voedger/voedger/pkg/iextengine"

// от sys сюда все попадает
// common - это типа sys, типа stateless, т.е. все то, что общее на все приложения и не зависит от текущего приложения
func ProvideExtensionEngineFactory(appFuncs iextengine.BuiltInAppExtFuncs, statelessFuncs iextengine.BuiltInExtFunc) iextengine.IExtensionEngineFactory {
	return extensionEngineFactory{appFuncs}
}

