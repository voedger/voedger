/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package iextengine

import (
	"context"
	"fmt"

	"github.com/juju/errors"
	"github.com/voedger/voedger/pkg/appdef"
)

var NullExtensionEngine IExtensionEngine = nullExtensionEngine{}

var NullExtensionEngineFactory IExtensionEngineFactory = nullExtensionEngineFactory{}

type nullExtensionEngine struct{}

func (nullExtensionEngine) SetLimits(limits ExtensionLimits) {}

func (nullExtensionEngine) Invoke(_ context.Context, n appdef.FullQName, _ IExtensionIO) error {
	return fmt.Errorf("unable nullExtensionEngine.Invoke(%v): %w", n, errors.NotSupported)
}

func (nullExtensionEngine) Close(context.Context) {}

type nullExtensionEngineFactory struct{}

func (nullExtensionEngineFactory) New(_ context.Context, _ appdef.AppQName, _ []ExtensionModule, _ *ExtEngineConfig, numEngines uint) ([]IExtensionEngine, error) {
	ee := make([]IExtensionEngine, numEngines)
	for i := uint(0); i < numEngines; i++ {
		ee[i] = NullExtensionEngine
	}
	return ee, nil
}
