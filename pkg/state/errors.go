/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"errors"
)

var ErrNotSupported = errors.New("not supported")
var ErrNotFound = errors.New("not found")
var ErrNotExists = errors.New("not exists")
var ErrExists = errors.New("exists")
var ErrIntentsLimitExceeded = errors.New("intents limit exceeded")
var ErrUnknownStorage = errors.New("unknown storage")
var ErrGetBatchNotSupportedByStorage = errors.New("get batch not supported by storage")
var ErrReadNotSupportedByStorage = errors.New("read not supported by storage")
var ErrUpdateNotSupportedByStorage = errors.New("update not supported by storage")
var ErrInsertNotSupportedByStorage = errors.New("insert not supported by storage")
var errTest = errors.New("test")
var errCurrentValueIsNotAnArray = errors.New("current value is not an array")
var errFieldByNameIsNotAnObjectOrArray = errors.New("field by name is not an object or array")
var errFieldByIndexIsNotAnObjectOrArray = errors.New("field by index is not an object or array")
var errNotImplemented = errors.New("not implemented")
