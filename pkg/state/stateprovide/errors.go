/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package stateprovide

import (
	"errors"
	"fmt"
)

var ErrNotSupported = errors.New("not supported")
var ErrNotExists = errors.New("not exists")
var ErrExists = errors.New("exists")

var ErrIntentsLimitExceeded = errors.New("intents limit exceeded")
var ErrUnknownStorage = errors.New("unknown storage")
var ErrGetNotSupportedByStorage = errors.New("get not supported by storage")
var ErrReadNotSupportedByStorage = errors.New("read not supported by storage")
var ErrUpdateNotSupportedByStorage = errors.New("update not supported by storage")
var ErrInsertNotSupportedByStorage = errors.New("insert not supported by storage")

var errCommandPrepareArgsNotSupportedByState = fmt.Errorf("%w: CommandPrepareArgs available for commands only", errors.ErrUnsupported)
var errQueryPrepareArgsNotSupportedByState = fmt.Errorf("%w: QueryPrepareArgs available for queries only", errors.ErrUnsupported)
var errQueryCallbackNotSupportedByState = fmt.Errorf("%w: QueryCallback available for queries only", errors.ErrUnsupported)
var errTest = errors.New("test")
