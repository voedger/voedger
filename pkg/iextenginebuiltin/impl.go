/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package iextenginebuiltin

import (
	"context"
	"fmt"

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
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("extension panic: %v", r)
		}
	}()
	if f, ok := e.funcs[extName]; ok {
		return f(ctx, io)
	}
	return undefinedExtension(extName.String())
}

func (e extensionEngine) Close(ctx context.Context) {}

func (f extensionEngineFactory) New(_ context.Context, _ []iextengine.ExtensionPackage, _ *iextengine.ExtEngineConfig, numEngines int) (result []iextengine.IExtensionEngine, err error) {
	result = make([]iextengine.IExtensionEngine, numEngines)
	for i := 0; i < numEngines; i++ {
		result[i] = &extensionEngine{f.funcs}
	}
	return
}
