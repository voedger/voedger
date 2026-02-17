/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import "time"

const (
	// how long Write() waits before deciding there is no consumer
	// e.g. if router is busy writing to the slow http client
	noConsumerTimeout = 10 * time.Second

	// how often to warn if the first response is not received yet
	firstResponseWaitWarningInterval = time.Minute
)
