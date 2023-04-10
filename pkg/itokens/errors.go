/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package itokens

import "errors"

var (
	ErrInvalidPayload          = errors.New("invalid payload")
	ErrSignerError             = errors.New("signer error")
	ErrTokenExpired            = errors.New("token expired")
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidAudience         = errors.New("invalid token audience")
)
