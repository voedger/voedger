/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

var ErrNotSupported = errors.New("not supported")
var ErrNotFound = errors.New("not found")
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
var errEntityRequiredForValueBuilder = errors.New("entity required for ValueBuilder")
var errWorkspaceDescriptorNotFound = errors.New("WorkspaceDescriptor not found in workspace")
var errDescriptorForUndefinedWorkspace = errors.New("workspace descriptor for undefined workspace")
var errCommandNotSpecified = errors.New("command not specified")
var errBlobIDNotSpecified = errors.New("blob ID not specified")
var ErrQNameIsNotDefinedInWorkspace = errors.New("qname is not defined in workspace")

func errUnexpectedType(actual interface{}) error {
	return fmt.Errorf("unexpected type: %v", actual)
}

func errIndexOutOfBounds(index int) error {
	return fmt.Errorf("index out of bounds: %d", index)
}

func typeIsNotDefinedInWorkspaceWithDescriptor(typ, ws appdef.QName) error {
	return fmt.Errorf("%s %w %s", typ.String(), ErrQNameIsNotDefinedInWorkspace, ws.String())
}
