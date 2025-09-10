/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
	"errors"
	"net/http"
)

var (
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
	errHTTPStatus503        = errors.New(http.StatusText(http.StatusServiceUnavailable))
)
