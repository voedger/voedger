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

var actualizationTimeoutLimit = 5 * time.Second

// waitForActualization waits for actualization to complete by repeatedly calling Start
func WaitForStart(t *testing.T, seq ISequencer, wsKind WSKind, wsID WSID) PLogOffset {
	timeoutCtx, timeoutCtxCancel := context.WithTimeout(context.Background(), actualizationTimeoutLimit)
	defer timeoutCtxCancel()

	offset, ok := seq.Start(wsKind, wsID)
	for !ok && timeoutCtx.Err() == nil {
		offset, ok = seq.Start(wsKind, wsID)
	}
	require.True(t, ok)

	return offset
}
