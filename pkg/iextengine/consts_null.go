/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package iextengine

import (
	"context"

	"github.com/juju/errors"
	"github.com/voedger/voedger/pkg/appdef"
)

var NullExtensionEngine IExtensionEngine = nullExtensionEngine{}

var NullExtensionEngineFactory IExtensionEngineFactory = nullExtensionEngineFactory{}

type nullExtensionEngine struct{}

func (nullExtensionEngine) SetLimits(limits ExtensionLimits) {}

func (nullExtensionEngine) Invoke(context.Context, appdef.FullQName, IExtensionIO) error {
	return errors.NotSupported
}

func (nullExtensionEngine) Close(context.Context) {}

type nullExtensionEngineFactory struct{}

func (nullExtensionEngineFactory) New(_ context.Context, _ []ExtensionPackage, _ *ExtEngineConfig, numEngines int) ([]IExtensionEngine, error) {
	ee := make([]IExtensionEngine, numEngines)
	for i := 0; i < numEngines; i++ {
		ee[i] = NullExtensionEngine
	}
	return ee, nil
}
