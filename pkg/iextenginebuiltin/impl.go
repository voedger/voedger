/*
  - Copyright (c) 2023-present unTill Software Development Group B. V.
    @author Michael Saigachenko
*/
package iextenginebuiltin

import (
	"context"

	"github.com/voedger/voedger/pkg/iextengine"
)

type extensionEngineFactory struct {
	funcs iextengine.BuiltInExtFuncs
}

type extensionEngine struct {
	funcs iextengine.BuiltInExtFuncs
}

func (e extensionEngine) SetLimits(limits iextengine.ExtensionLimits) {}

func (e extensionEngine) Invoke(ctx context.Context, extName iextengine.ExtQName, io iextengine.IExtensionIO) (err error) {
	if f, ok := e.funcs[extName]; ok {
		return f(ctx, io)
	}
	return undefinedExtension(extName.String())
}

func (e extensionEngine) Close() {}

func (f extensionEngineFactory) New(packageNameToLocalPath map[string]string, config *iextengine.ExtEngineConfig, numEngines int) (result []iextengine.IExtensionEngine) {
	result = make([]iextengine.IExtensionEngine, numEngines)
	for i := 0; i < numEngines; i++ {
		result[i] = &extensionEngine{f.funcs}
	}
	return
}
