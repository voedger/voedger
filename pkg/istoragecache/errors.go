/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package istoragecache

import "errors"

var (
	errKeyLengthExceeded             = errors.New("key length exceeded")
	errKeyPoolReachedMaximumCapacity = errors.New("key pool reached maximum capacity")
)
