/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import "time"

const (
	// how long Write() waits before returning ErrSendResponseTimeout
	// happens when router is busy writing to the slow http client
	sendResponseTimeout = 10 * time.Second

	// how often to warn if the first response is not received yet
	firstResponseWaitWarningInterval = time.Minute
)
