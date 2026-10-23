/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package commandprocessor

import "errors"

var (
	errRecoveryInProgress        = errors.New("recovery is in progress")
	errRecoveryLimit             = errors.New("recovery concurrency limit is reached")
	errRecoveryFailed            = errors.New("recovery failed")
	errInvalidRecoveryCheckpoint = errors.New("invalid recovery checkpoint")
)
