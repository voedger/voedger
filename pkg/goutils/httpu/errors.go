/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"errors"
)

var (
	// ErrRetryExhausted marks a failure returned after its status retry duration expires.
	ErrRetryExhausted       = errors.New("retry duration exceeded")
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
	errRetry                = errors.New("retry")
)
