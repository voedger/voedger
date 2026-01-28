/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */
package storage

import "errors"

var (
	// ErrAppTTLValidation is the category error for all IAppTTLStorage validation errors.
	// Use errors.Is(err, ErrAppTTLValidation) to check if error is a validation error.
	ErrAppTTLValidation = errors.New("app TTL storage validation error")

	ErrKeyEmpty       = errors.New("key is empty")
	ErrKeyTooLong     = errors.New("key exceeds maximum length")
	ErrKeyInvalidUTF8 = errors.New("key is not valid UTF-8")
	ErrValueTooLong   = errors.New("value exceeds maximum length")
	ErrInvalidTTL     = errors.New("TTL must be between 1 and 31536000 seconds")
)
