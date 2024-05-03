/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package btstrp

import "time"

const (
	retryOnHTTPErrorTimeout = 15 * time.Second
	retryOnHTTPErrorDelay   = time.Second
)
