/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package journal

import "errors"

var (
	errIndexNotSupported                        = errors.New("index not supported")
	errArgumentTypeNotSupported                 = errors.New("argument type not supported")
	errOffsetMustBePositive                     = errors.New("offset must be positive")
	errFromOffsetMustBeLowerOrEqualToTillOffset = errors.New("'from' offset must be lower or equal to 'till' offset")
)
