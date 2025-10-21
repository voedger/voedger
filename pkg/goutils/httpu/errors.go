/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"errors"
)

var (
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
	errRetry                = errors.New("retry")
)
