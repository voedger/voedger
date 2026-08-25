/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package appparts

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartitionBorrowRetryPolicy(t *testing.T) {
	require := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	appParts, cleanup, err := newAppPartitions(
		ctx,
		nil,
		NullSyncActualizerFactory,
		NullActualizerRunner,
		NullSchedulerRunner,
		NullExtensionEngineFactories,
		nil,
	)
	require.NoError(err)
	defer func() {
		cancel()
		cleanup()
	}()

	onError := appParts.(*apps).partBorrowRetryCfg.OnError
	for kind := range ProcessorKind_Count {
		t.Run(kind.TrimString(), func(t *testing.T) {
			opErr := errNotAvailableEngines[kind]
			retry, abortErr := onError(1, 0, opErr)
			require.True(retry)
			require.NoError(abortErr)
		})
	}

	otherErr := errors.New("other borrow error")
	retry, abortErr := onError(1, 0, otherErr)
	require.False(retry)
	require.ErrorIs(abortErr, otherErr)
}
