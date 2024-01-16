/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package ibus

import "errors"

var (
	// ErrTimeoutExpired s.e.
	ErrTimeoutExpired = errors.New("timeout expired")

	// ErrNoConsumer shows that consumer of further sections is gone. Further sections sending is senceless.
	ErrNoConsumer = errors.New("no consumer for the stream")
)
