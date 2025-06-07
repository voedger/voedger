/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package bus

import "errors"

var (
	ErrSendTimeoutExpired = errors.New("send timeout expired")
	ErrNoConsumer         = errors.New("no consumer for the stream")
)
