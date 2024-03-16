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
var errTest = errors.New("test")
var errCurrentValueIsNotAnArray = errors.New("current value is not an array")
var errFieldByNameIsNotAnObjectOrArray = errors.New("field by name is not an object or array")
var errFieldByIndexIsNotAnObjectOrArray = errors.New("field by index is not an object or array")
var errNotImplemented = errors.New("not implemented")
var errWorkspaceDescriptorNotFound = errors.New("WorkspaceDescriptor not found in workspace")
var errWorkspaceUndefined = errors.New("workspace undefined")

func errUndefined(name string) error {
	return errors.New(name + " undefined")
}

func errIndexOutOfBounds(index int) error {
	return fmt.Errorf("index out of bounds: %d", index)
}

func typeIsNotDefinedInWorkspace(typ, ws appdef.QName) error {
	return fmt.Errorf("%s is not available in workspace %s", typ.String(), ws.String())
}
