/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package appparts

import (
	"context"

	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/istructs"
)

func NewTestAppParts(asp istructs.IAppStructsProvider) (res IAppPartitions, cleanup func()) {
	vvmCtx, cancel := context.WithCancel(context.Background())
	res, clean, err := New2(
		vvmCtx,
		asp,
		NullSyncActualizerFactory,
		NullActualizerRunner,
		NullSchedulerRunner,
		NullExtensionEngineFactories,
		iratesce.TestBucketsFactory,
	)
	if err != nil {
		panic(err)
	}
	return res, func() {
		cancel()
		clean()
	}
}
