/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package appparts

import (
	"context"

	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
)

func NewTestAppParts(asp istructs.IAppStructsProvider) (IAppPartitions, func()) {
	vvmCtx, cancel := context.WithCancel(context.Background())
	appParts, cleanup, err := New2(
		vvmCtx,
		asp,
		NullSyncActualizerFactory,
		NullActualizerRunner,
		NullSchedulerRunner,
		NullExtensionEngineFactories,
		irates.NullBucketsFactory,
	)
	if err != nil {
		panic(err)
	}
	combinedCleanup := func() {
		cancel()
		cleanup()
	}
	return appParts, combinedCleanup
}
