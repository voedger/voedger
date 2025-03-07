/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import "errors"

var (
	ErrFieldsMissed          = errors.New("fields are missed")
	ErrFieldTypeMismatch     = errors.New("field type mismatch")
	ErrUnexpectedStatusCode  = errors.New("unexpected status code")
	ErrNumberOverflow        = errors.New("number overflow")
	ErrRetryAttemptsExceeded = errors.New("retry attempts exceeded")
)
