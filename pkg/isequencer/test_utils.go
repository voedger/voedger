/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package isequencer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var actualizationTimeoutLimit = 100 * time.Millisecond

// waitForActualization waits for actualization to complete by repeatedly calling Start
func WaitForStart(t *testing.T, seq ISequencer, wsKind WSKind, wsID WSID, shouldBeOk bool) PLogOffset {
	timeoutCtx, timeoutCtxCancel := context.WithTimeout(context.Background(), actualizationTimeoutLimit)
	defer timeoutCtxCancel()

	offset, ok := seq.Start(wsKind, wsID)
	for !ok && timeoutCtx.Err() == nil {
		offset, ok = seq.Start(wsKind, wsID)
	}

	if shouldBeOk {
		require.True(t, ok)
	} else {
		require.False(t, ok)
	}

	return offset
}
